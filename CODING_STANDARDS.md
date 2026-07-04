# Coding Standards

This project follows the architectural patterns from *Let's Go* and *Let's Go Further* by Alex Edwards (reference copies: `docs/references/lets-go.html/` and `docs/references/lets-go-further.html/`), adapted for this project's two-binary shape, templ/templui instead of `html/template`, and Postgres (Neon) instead of MySQL. *Let's Go Further* builds a JSON API (the "Greenlight" movies app) — anything specific to JSON responses is ignored; the operational patterns (connection pooling, migrations, advanced CRUD, pagination, rate limiting, graceful shutdown, metrics, build/release) apply regardless of response format. Where this document contradicts either book, this document wins — the deviations are deliberate and explained below.

Component reference: `docs/references/templui/llms.txt` documents the templui component catalog.

## Project structure

Two executables sharing one `internal/` tree:

```
/
├── cmd/
│   ├── blog/            # public binary — read-only, internet-facing (Cloudflare Tunnel)
│   │   ├── main.go
│   │   ├── routes.go
│   │   ├── handlers.go
│   │   ├── middleware.go
│   │   └── helpers.go
│   └── blog-admin/      # admin binary — compose/edit, tailnet-only
│       ├── main.go
│       ├── routes.go
│       ├── handlers.go
│       ├── middleware.go
│       └── helpers.go
├── internal/
│   ├── models/          # Post, Project, Tag — Postgres-backed, shared by both binaries
│   ├── validator/
│   └── assert/          # test-only helper package
└── ui/
    ├── templ/
    │   ├── layout/       # shared shell/nav (templui-based)
    │   ├── pages/blog/   # home, post view, project view, about
    │   └── pages/admin/  # compose, edit
    └── static/           # CSS, vendored Alpine.js, images — embedded via embed.FS
```

Each `cmd/*` package holds *application-specific* code only. `internal/` holds reusable, non-app-specific code and is import-restricted by the Go toolchain to this module — nothing outside the repo can import it, even though the repo is public.

Use a locally-scoped `http.NewServeMux()` in each binary's `routes()`. Never rely on `http.DefaultServeMux`.

## Routing

- Go's stdlib mux: method-prefixed patterns (`"GET /{$}"`), wildcards `{slug}`, `{$}` to stop a trailing-slash pattern from acting as a catch-all.
- Most-specific-pattern-wins; keep overlapping patterns to a minimum (ambiguous overlaps panic at startup).
- Validate every wildcard before use, same shape as the book: parse/lookup, and 404 on failure — never let an invalid path value reach a model call.
- Handler naming: `postView` (GET) / `postCreate` (GET, admin) / `postCreatePost` (POST, admin).
- `routes()` returns `http.Handler` (post-middleware), not `*http.ServeMux`.

## Dependency injection

One `application` struct per binary, holding that binary's dependencies; handlers are methods on it. No globals.

```go
// cmd/blog/main.go
type application struct {
    logger  *slog.Logger
    posts   models.PostModelInterface
    projects models.ProjectModelInterface
}

// cmd/blog-admin/main.go
type application struct {
    logger         *slog.Logger
    posts          models.PostModelInterface
    projects       models.ProjectModelInterface
    formDecoder    *form.Decoder
    sessionManager *scs.SessionManager // flash messages only — see Sessions
}
```

Never stash long-lived dependencies (DB pool, logger) in request context — only request-scoped data belongs there.

## Configuration

Use the `flag` package (`flag.String("addr", ":4000", "...")`, `flag.Parse()`) rather than bare `os.Getenv` — gives type conversion, defaults, and free `-help`. Pipe environment values (Neon DSN, etc.) in as flag defaults if the deployment needs it.

## Logging

`log/slog` in both binaries: `slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{...}))`. Structured key-value logging (`logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())`), never `log.Fatal` — log at Error then `os.Exit(1)`. Route the `http.Server`'s own `ErrorLog` through slog via `slog.NewLogLogger(...)`.

## Error handling

Centralize in each binary's `helpers.go`:

```go
func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
    app.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
    http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
func (app *application) clientError(w http.ResponseWriter, status int) {
    http.Error(w, http.StatusText(status), status)
}
```

Model-layer sentinel errors (`models.ErrNoRecord`, etc.) checked with `errors.Is`. Panic only for genuine programmer errors, never for expected/operational failures.

## Middleware

Standard closure shape: `func mw(next http.Handler) http.Handler { return http.HandlerFunc(func(w, r *http.Request) { ...; next.ServeHTTP(w, r) }) }`. Use `justinas/alice` for composable chains.

Both binaries:
- `standard` chain (wraps the whole mux): `recoverPanic`, `logRequest`, `commonHeaders`.

`blog-admin` additionally:
- `dynamic` chain (wraps non-static routes): `sessionManager.LoadAndSave`, `preventCSRF` (see CSRF below).

`blog` has no `dynamic` chain — it's read-only, no sessions, no CSRF surface.

## Database models pattern

`internal/models/<entity>.go`: a plain struct for the row, a `*Model` struct wrapping the pool, and an interface for mocking in tests — same shape as the book, Postgres instead of MySQL:

```go
type Post struct {
    ID        int
    Title     string
    Slug      string
    Body      string
    SoWhat    string
    Version   int  // optimistic concurrency — see below
    Published time.Time
}
type PostModel struct{ DB *pgxpool.Pool }
type PostModelInterface interface {
    Insert(p Post) (int, error)
    Get(slug string) (Post, error)
    Latest() ([]Post, error)
    Update(p Post) error
}
```

`openDB(dsn string)` helper in each `main.go`, pool created once, injected via `application`, closed with `defer`. `internal/models/errors.go` holds sentinel errors; translate Postgres-specific errors (e.g. unique-violation) into these before returning.

**Connection pool.** Always set explicit limits rather than relying on defaults — start with `MaxOpenConns=25`, `MaxIdleConns=25` (`<= MaxOpenConns`), `ConnMaxIdleTime=15*time.Minute`, exposed as flags so they can be tuned per-binary without a rebuild. Verify the pool with a bounded `PingContext` (~5s timeout) at startup — an unreachable DB should fail fast, not silently. Use Neon's **direct (unpooled)** connection string, not its PgBouncer pooler endpoint — both binaries are long-running processes managing their own pool, not serverless functions making one-shot connections, so Neon's pooler adds nothing here and can complicate prepared-statement behavior.

**Query timeouts.** Every model method creates its own bounded context around the query, not the inbound request context: `ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()`. Applied uniformly to `Insert`/`Get`/`Update`/`Latest`.

**Optimistic concurrency.** Since posts are freely editable (see the ADR-implied product decision), every mutable table gets a `version int NOT NULL DEFAULT 1` column to guard against lost updates from concurrent edits:

```sql
UPDATE posts SET title=$1, body=$2, so_what=$3, version=version+1
WHERE id=$4 AND version=$5
RETURNING version
```

`sql.ErrNoRows` from that `Scan` means the record was edited or deleted since it was loaded — map to `models.ErrEditConflict` and return `409 Conflict` from the handler, never silently overwrite.

**SQL migrations.** Paired up/down `.sql` files under `/migrations`, sequentially numbered (`000001_create_posts_table.up.sql` / `.down.sql`), managed with `golang-migrate` — this directory is the single source of schema truth and is committed to version control (not gitignored, unlike `docs/references/`). Conventions: `bigint GENERATED ALWAYS AS IDENTITY` for primary keys, `NOT NULL` + a sensible `DEFAULT` on every column, `text` instead of `varchar(n)`, `CHECK` constraints for business rules in their own migration, `IF EXISTS`/`IF NOT EXISTS` guards throughout. Migrations run explicitly via a Makefile target, never automatically from binary startup.

## Filtering, sorting, pagination

Applies to any listing endpoint (`blog`'s home page / Project page, `blog-admin`'s post list):

- Shared `Filters` struct (`Page`, `PageSize`, `Sort`, `SortSafelist []string`), validated: `Page` capped well below any realistic post count, `PageSize` capped at 100, `Sort` checked against `SortSafelist` via `validator.PermittedValue`.
- **Never interpolate raw sort input into SQL.** The only place `fmt.Sprintf` is acceptable for building a query is injecting a column/direction that has already been checked against `SortSafelist` — placeholders can't parameterize identifiers.
- Always add a secondary `ORDER BY ..., id ASC` — Postgres doesn't guarantee stable order without a unique tiebreaker, which matters once pagination is involved.
- Pagination via `LIMIT`/`OFFSET`; get the total count in the same query via a window function (`SELECT count(*) OVER(), ... LIMIT $n OFFSET $m`) rather than a separate `COUNT(*)` query.

## Forms & validation

`internal/validator` package, embedded in every form struct: `CheckField`, `AddFieldError`, `AddNonFieldError`, `Valid()`, plus `NotBlank`, `MaxChars`, `MinChars`, `PermittedValue[T comparable]`.

Decode POST bodies with `go-playground/form` via `app.decodePostForm(r, &form)` rather than manual `r.PostForm.Get(...)` parsing.

On validation failure: re-render the same page with **422 Unprocessable Entity**, passing the form struct (values + `FieldErrors`) as a typed parameter into the templ component — same idea as the book's `templateData.Form any` field, but as an explicit typed argument to the component function rather than a template action.

## CSRF (blog-admin only)

`blog-admin` has state-changing forms (compose, edit) and must be protected even though it's tailnet-only — Tailscale prevents *unauthorized network access*, not a malicious page in your browser submitting a forged POST while you're on the tailnet.

Deliberate deviation from the book: use the modern stdlib approach instead of adding `justinas/nosurf` as a dependency —

- `scs` already sets `SameSite=Lax` on the session cookie by default — keep it.
- Add `http.CrossOriginProtection` middleware (Go stdlib) to the `dynamic` chain, checking `Sec-Fetch-Site`/`Origin`.
- No CSRF token field needed in forms with this approach (unlike `nosurf`'s hidden `csrf_token` input).

This is simpler and dependency-free, appropriate for a single-user admin tool that doesn't need to support pre-2020 browsers.

## Sessions (blog-admin only, flash messages only)

Per the network-boundary decision (ADR-0001), `blog-admin` has **no login system** — reaching it over Tailscale is the authentication. `scs` sessions exist solely for flash messages (e.g. "Post published"), not identity:

```go
sessionManager := scs.New()
sessionManager.Store = <driver>store.New(db)
sessionManager.Lifetime = 12 * time.Hour
```

`Put(ctx, "flash", msg)` on write, `PopString(ctx, "flash")` in a `newTemplateData`-equivalent helper so every render surfaces and clears it. No `RenewToken`, no `authenticatedUserID`, no `requireAuthentication` middleware — there is no authenticated-vs-anonymous distinction to make.

`blog` has no sessions at all.

## Templates (templ / templui)

templ replaces `html/template` entirely — no runtime parsing, no template cache to build. `templ generate` compiles `.templ` files to Go at build time; the compiled binary is self-contained.

Handlers call component functions directly and render:

```go
func (app *application) postView(w http.ResponseWriter, r *http.Request) {
    post, err := app.posts.Get(r.PathValue("slug"))
    if err != nil { ... app.serverError(...); return }
    pages.PostView(post).Render(r.Context(), w)
}
```

Shared layout/nav lives under `ui/templ/layout/` and is composed into every page component, mirroring the book's base-template inheritance but via ordinary Go function composition instead of `{{define}}`/`{{template}}` actions.

## Client-side interactivity (Alpine.js / Alpine AJAX)

Default is plain server-rendered HTML: standard `<form>` posts, full-page navigations. No JS is added by default.

- **Alpine.js**: only for small, local, ephemeral UI state that doesn't need the server at all — a mobile nav toggle, a client-side character counter on the So What field. Written inline via `x-data` in the templ markup, no build step.
- **Alpine AJAX**: only when a specific interaction genuinely benefits from a partial-page update (e.g. filtering posts by tag/Project without a full reload) — and only *after* the plain-HTML, full-reload version of that same endpoint already works. Alpine AJAX enhances the existing route/fragment; it never replaces the no-JS path.
- Vendor both into `ui/static/` (embedded, same as CSS) rather than pulling from a CDN — consistent with self-hosting everything else.

If a feature doesn't need either, don't add either.

## Server config & timeouts

Construct `*http.Server` explicitly in both binaries instead of `http.ListenAndServe`, with `IdleTimeout`, `ReadTimeout`, `WriteTimeout` always set explicitly (mitigates Slowloris-style slow-client issues; `IdleTimeout` doesn't default from `ReadTimeout`).

Deliberate deviation from the book: neither binary terminates TLS itself. `blog`'s TLS is terminated at Cloudflare Tunnel; `blog-admin`'s transport security comes from the Tailscale (WireGuard) network layer. Both binaries serve plain HTTP locally. The book's self-signed-cert/TLS-config chapter (09.03–09.05) doesn't apply here.

**Graceful shutdown.** Both binaries (running as long-lived homelab services) catch `SIGINT`/`SIGTERM` on a buffered `chan os.Signal, 1`, then call `srv.Shutdown(ctx)` with a bounded context (~30s) and exit cleanly rather than dropping in-flight requests. `http.ErrServerClosed` from `ListenAndServe` is the expected/good outcome, not an error to log. Server construction lives in its own `server.go`/`serve()` method, not inline in `main()`.

## Rate limiting (`blog` primarily)

`blog` is internet-facing and should defend against scraping/abuse; `blog-admin` is tailnet-only so this is lower priority there but cheap to share.

- Per-client token bucket via `golang.org/x/time/rate`: a `map[string]*client{limiter, lastSeen}` keyed by IP, guarded by a `sync.Mutex` (unlocked explicitly before calling `next.ServeHTTP`, not deferred).
- A background goroutine sweeps the map every minute, evicting entries older than a few minutes, to bound memory.
- Resolve the real client IP via a real-IP helper that checks `X-Forwarded-For`/`X-Real-IP` before falling back to `r.RemoteAddr` — necessary since requests arrive via Cloudflare Tunnel, not directly.
- Configurable via flags (`rps`, `burst`, `enabled`) so it can be disabled for local dev/load testing without a code change.
- Note this in-memory approach only works for a single instance — fine here since there's exactly one `blog` process, but wouldn't survive a move to multiple replicas without an external store.

## Metrics (ops, not page analytics)

Separate from the self-hosted Umami/Plausible *page-view* analytics (Q13 of the design) — these are internal operational metrics, exposed via `expvar` for your own debugging, not visitor tracking:

- Mount `expvar.Handler()` at `/debug/vars`, but never expose it on `blog` (internet-facing) without access control — it can leak the DSN via cmdline args and is a DoS target. On `blog-admin` it's fine as-is since the route is already tailnet-only.
- Register request-level counters via middleware wrapping the whole router: `total_requests_received`, `total_responses_sent`, cumulative processing time, and `total_responses_sent_by_status` (via a small `http.ResponseWriter` wrapper that also implements `Unwrap() http.ResponseWriter`). All as `expvar.Int`/`expvar.Map`, updated with `.Add(n)` — safe for concurrent use without extra locking.
- Useful `expvar.Publish` values: `runtime.NumGoroutine()`, `db.Stats()` (pool health), current version.

## File embedding

Static assets (`ui/static/`) embedded the same way as the book:

```go
package ui
import "embed"
//go:embed "static"
var Files embed.FS
```

Served via `http.FileServerFS(ui.Files)`. No template embedding needed (templ compiles to Go, not parsed at runtime).

## Testing

- `*_test.go` colocated with code; table-driven tests via anonymous-struct slices + `t.Run` sub-tests.
- `internal/assert` helper package (`Equal`, `NotEqual`, `True`, `False`, `Nil`, `NotNil`), each calling `t.Helper()`.
- Unit-test handlers/middleware with `httptest.NewRecorder()` + `http.NewRequest(...)`.
- `blog-admin` end-to-end tests: a `testServer` wrapping `httptest.NewServer(app.routes())` (no TLS needed locally per the deviation above) with `get()`/`postForm()` helpers and a cookie jar for session persistence; reset the jar between sub-tests.
- Mock model dependencies in `internal/models/mocks` implementing `PostModelInterface`/`ProjectModelInterface` with fixture data.
- Integration tests against a real test Postgres (Neon branch or local instance): `newTestDB(t)` running `testdata/setup.sql`, `t.Cleanup` running teardown; skip via `testing.Short()`.

## Build & release

**Makefile** at the repo root, targets namespaced with `/` (`db/migrations/up`, `run/blog`, `run/blog-admin`, `build/blog`), never `:` in target names. All action-only rules marked `.PHONY`. A `confirm` prerequisite guards destructive targets (e.g. running migrations against a real DB): `@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]`. A self-documenting `help` target (parses `## target: description` comments from the Makefile itself) is the default (first) rule, so bare `make` prints usage.

**Environment variables.** No secrets or DSNs hardcoded in `main.go` — flags default to `""`, and a gitignored `.envrc` (added to `.gitignore` the moment it's created) supplies real values via `include .envrc` in the Makefile, injected as `${VAR}` into `make run/...` targets. Flags remain the only way the running binary is actually configured; env vars are just a dev-time convenience for populating them.

**Quality control**, run via `make audit` before committing (distinct from the existing `commitizen` pre-commit hook, which only lints commit messages): `go mod tidy -diff`, `go mod verify`, `go vet ./...`, `go tool staticcheck ./...`, `go test -race -vet=off ./...`. A separate mutating `make tidy` runs `go mod tidy`, `go fix ./...`, `go fmt ./...`. Install `staticcheck` as a tool dependency in `go.mod` (`go get -tool honnef.co/go/tools/cmd/staticcheck@latest`), not a separately managed global binary.

**Building binaries.** `go build -ldflags='-s' -o=./bin/... ./cmd/...` (strips symbol table, smaller binary); cross-compile explicitly for the homelab's actual OS/arch via `GOOS`/`GOARCH` in addition to any local dev build. `bin/` is gitignored — never commit built binaries. Derive `version` from VCS metadata (`internal/vcs.Version()` via `debug.ReadBuildInfo()`) rather than a hardcoded constant, so it's automatically the Go pseudo-version or exact tag (suffixed `+dirty` on uncommitted changes) — this only populates for `go build`, not `go run`. A `-version` flag prints it and exits immediately after `flag.Parse()`, before any DB/server setup.
