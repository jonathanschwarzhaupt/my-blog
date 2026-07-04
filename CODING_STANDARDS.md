# Coding Standards

This project follows the architectural patterns from *Let's Go* and *Let's Go Further* by Alex Edwards (reference copies: `docs/references/lets-go.html/` and `docs/references/lets-go-further.html/`), adapted for this project's two-binary shape, templ/templui instead of `html/template`, and Postgres (Neon) instead of MySQL. *Let's Go Further* builds a JSON API (the "Greenlight" movies app) — anything specific to JSON responses is ignored; the operational patterns (connection pooling, migrations, advanced CRUD, pagination, rate limiting, graceful shutdown, metrics, build/release) apply regardless of response format. Where this document contradicts either book, this document wins — the deviations are deliberate and explained below.

Component reference: `docs/references/templui/llms.txt` documents the templui component catalog.

The database layer (sqlc + goose, `internal/database` generated / `internal/models` hand-written) and the `options.go` config pattern follow `jonathanschwarzhaupt/go-cookbook`'s established convention, not the books' hand-rolled model / inline-flags approach.

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
│   ├── blog-admin/      # admin binary — compose/edit, tailnet-only
│   │   ├── main.go
│   │   ├── routes.go
│   │   ├── handlers.go
│   │   ├── middleware.go
│   │   └── helpers.go
│   └── migrate/         # standalone goose runner — also the Kubernetes init-container image
│       └── main.go
├── internal/
│   ├── database/        # sqlc-generated code only — never hand-edited (`sqlc generate`)
│   ├── models/           # hand-written domain types (Post, Project, Tag) + row-mapping helpers
│   ├── validator/
│   └── assert/          # test-only helper package
├── sql/
│   ├── schema/           # goose migrations (embedded into cmd/migrate)
│   └── queries/          # sqlc query files (source for internal/database)
├── sqlc.yaml
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
    logger *slog.Logger
    db     database.Querier // sqlc-generated interface (internal/database), see Database section
}

// cmd/blog-admin/main.go
type application struct {
    logger         *slog.Logger
    db             database.Querier
    formDecoder    *form.Decoder
    sessionManager *scs.SessionManager // flash messages only — see Sessions
}
```

Never stash long-lived dependencies (DB pool, logger) in request context — only request-scoped data belongs there.

## Configuration

Matches the `options.go` pattern from `jonathanschwarzhaupt/go-cookbook`: each binary gets its own `options` struct + `parseOptions() *options` in `options.go`, keeping `main.go` itself down to wiring, not flag declarations.

```go
// cmd/blog/options.go
type options struct {
    addr           string
    dbDSN          string
    dbMaxConns     int
    dbMinConns     int
    dbMaxIdleTime  time.Duration
    displayVersion bool
}

func parseOptions() *options {
    opts := &options{}
    flag.StringVar(&opts.addr, "addr", ":4000", "HTTP network address")
    flag.StringVar(&opts.dbDSN, "db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
    // ...
    flag.Parse()
    return opts
}
```

Use the `flag` package (gives type conversion, defaults, and free `-help`) rather than bare `os.Getenv` for every setting. Pipe environment values (Neon DSN, etc.) in as flag defaults so the same binary works identically whether launched via `make run/...` (env-backed default) or with an explicit override flag — don't make the DSN a hard-required env var with no flag path, since that removes the override needed for tests/local dev pointing at a different database.

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

## Database layer: sqlc + goose

Deliberate deviation from *Let's Go*'s hand-rolled `PostModel`/`PostModelInterface` pattern: **no hand-written SQL-calling model methods, and no ORM.** Two generators own this layer instead, matching the convention already proven in `jonathanschwarzhaupt/go-cookbook`:

- **goose** owns schema migrations (`sql/schema/`).
- **sqlc** owns queries, generating typed Go from them (`sql/queries/` → `internal/database/`).

### `internal/database` — generated, never hand-edited

`sqlc generate` (via `make sqlc/generate`) reads `sql/schema/*.sql` (for column types) and `sql/queries/*.sql` (annotated queries) and writes `internal/database/{db,models,*.sql.go}`. `sqlc.yaml` sets `sql_package: pgx/v5` and `emit_interface: true`, so sqlc emits both a concrete `*Queries` and a `Querier` interface — `Querier` is what gets mocked in handler tests, no hand-written interface needed:

```sql
-- sql/queries/posts.sql
-- name: GetPost :one
SELECT * FROM posts WHERE slug = $1;

-- name: InsertPost :one
INSERT INTO posts (title, slug, body, so_what, tags) VALUES ($1, $2, $3, $4, $5) RETURNING *;
```

```go
// application struct field, either binary
db database.Querier
```

Treat everything under `internal/database` as read-only generated output — changes go through `sql/queries/*.sql` + regenerate, never a direct edit.

### `internal/models` — hand-written domain types only

Holds plain domain structs (`Post`, `Project`, `Tag`) shaped for handlers/templates, plus small mapper functions converting a generated `database.Post` row into `models.Post` where the shapes diverge (e.g. hiding internal-only columns, converting `pgtype` wrappers to plain Go types). Handlers call `app.db.GetPost(ctx, slug)` directly and map the result — there is no `PostModel.Get` wrapper method to write or maintain.

`internal/models/errors.go` holds sentinel errors (`ErrNoRecord`, `ErrEditConflict`); map Postgres-specific errors (unique-violation, no-rows) into these at the point a handler receives them from `app.db`.

**Connection pool.** `internal/models` also keeps `OpenPool(ctx, dsn, cfg PoolConfig) (*pgxpool.Pool, error)` — the one piece of hand-written DB plumbing that isn't sqlc's concern. Always set explicit limits rather than relying on defaults. Verify the pool with a bounded `Ping` (~5s timeout) at startup — an unreachable DB should fail fast, not silently. Use Neon's **direct (unpooled)** connection string, not its PgBouncer pooler endpoint — both binaries are long-running processes managing their own pool, not serverless functions making one-shot connections.

Deliberate deviation from *Let's Go Further*: the book uses `github.com/lib/pq` (a `database/sql` driver), configured via `db.SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime`/`SetConnMaxIdleTime`. This project uses `github.com/jackc/pgx/v5/pgxpool` directly instead — no `database/sql` indirection, since the project is Postgres-only and pgx's native pool already pairs with sqlc's `sql_package: pgx/v5` codegen. The two pools don't expose identical knobs, so here's the explicit mapping (all four exposed as flags, per binary, tunable without a rebuild):

| Book (`database/sql` + `lib/pq`) | This project (`pgxpool`) | Book's value | Ours | Why |
|---|---|---|---|---|
| `MaxOpenConns` | `MaxConns` | 25 | 25 | Same reasoning — comfortably below Postgres' default 100-connection hard limit, headroom for two binaries sharing one Neon compute. |
| `MaxIdleConns` | *(no equivalent)* | 25 (== MaxOpenConns) | — | `pgxpool` has no separate idle-connection ceiling — it already keeps connections open up to `MaxConns` without a distinct idle cap, so the book's "set MaxIdleConns == MaxOpenConns" workaround is unnecessary here. |
| *(no equivalent)* | `MinConns` | — | 5 | `pgxpool`-only concept: a *proactive floor* the pool eagerly maintains, not an idle ceiling — semantically different from `MaxIdleConns`, so it isn't a straight numeric port. Kept modest (not 25) since a low-traffic personal blog doesn't need 25 warm connections held open at all times. |
| `ConnMaxLifetime` | `MaxConnLifetime` | unlimited | 1 hour | The book leaves this unlimited because their local dev Postgres has no compute-recycling concerns. Neon's serverless compute can suspend/resume, so this project sets an explicit finite lifetime (matching `pgxpool`'s own built-in default) rather than relying on an implicit library default — every pool setting should be visible in `PoolConfig`, not left to whatever the library defaults to. |
| `ConnMaxIdleTime` | `MaxConnIdleTime` | 15 minutes | 15 minutes | Same value, same reasoning — free up connections that aren't being reused. |

**Query timeouts.** Handlers create their own bounded context around each `app.db.*` call, parented on `context.Background()` rather than the inbound request context: `ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()`. This deliberately decouples the write from the client's connection — an admin closing a tab or a flaky connection dropping mid-request shouldn't cancel a DB write that's already in flight.

**Optimistic concurrency.** Since posts are freely editable, every mutable table gets a `version int NOT NULL DEFAULT 1` column to guard against lost updates from concurrent edits, expressed as a named sqlc query:

```sql
-- name: UpdatePost :one
UPDATE posts SET title=$1, body=$2, so_what=$3, version=version+1
WHERE id=$4 AND version=$5
RETURNING *;
```

`pgx.ErrNoRows` from that call means the record was edited or deleted since it was loaded — the handler maps this to `models.ErrEditConflict` and returns `409 Conflict`, never a silent overwrite.

### Migrations (goose)

Single-file migrations under `sql/schema/`, sequentially numbered (`00001_create_posts_table.sql`), each containing `-- +goose Up` / `-- +goose Down` sections — this directory is the single source of schema truth and is committed to version control (not gitignored, unlike `docs/references/`). Conventions: `bigint GENERATED ALWAYS AS IDENTITY` for primary keys, `NOT NULL` + a sensible `DEFAULT` on every column, `text` instead of `varchar(n)`, `CHECK` constraints for business rules, `IF EXISTS`/`IF NOT EXISTS` guards throughout.

`cmd/migrate` is a small standalone binary (goose used as a library, not its all-dialect CLI, to avoid pulling in every driver goose/golang-migrate support — MySQL, Cassandra, Vertica, etc. — for a Postgres-only project) that embeds `sql/schema` via `embed.FS` and runs `goose.Up`/`Down`/`Status` against a DSN flag. It's invoked locally via `make db/migrations/up`, and is built as its own minimal container image to run as a **Kubernetes init container** ahead of `blog`/`blog-admin` in the homelab — migrations are never run automatically from either server binary's startup path.

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
- Mock the database dependency by implementing sqlc's generated `database.Querier` interface with fixture data — no hand-written interface to maintain, since `emit_interface: true` generates it.
- Integration tests against a real test Postgres (Neon branch or local instance): `newTestDB(t)` running goose migrations from `sql/schema` against a scratch database, `t.Cleanup` tearing it down; skip via `testing.Short()`.

## Build & release

**Makefile** at the repo root, targets namespaced with `/` (`db/migrations/up`, `run/blog`, `run/blog-admin`, `build/blog`), never `:` in target names. All action-only rules marked `.PHONY`. A `confirm` prerequisite guards destructive targets (e.g. running migrations against a real DB): `@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]`. A self-documenting `help` target (parses `## target: description` comments from the Makefile itself) is the default (first) rule, so bare `make` prints usage.

**Environment variables.** No secrets or DSNs hardcoded in `main.go` — flags default to `""`, and a gitignored `.envrc` (added to `.gitignore` the moment it's created) supplies real values via `include .envrc` in the Makefile, injected as `${VAR}` into `make run/...` targets. Flags remain the only way the running binary is actually configured; env vars are just a dev-time convenience for populating them. **Don't quote values in `.envrc`** (`export VAR=value`, not `export VAR="value"`) — Make's `include` parses `export NAME = value` as native syntax rather than sourcing it through a shell, so quote characters become part of the value literally and corrupt anything that reads the env var directly (e.g. a Go test calling `os.Getenv`) rather than through a `${VAR}`-interpolated recipe line, where the shell strips them.

**Quality control**, run via `make audit` before committing (distinct from the existing `commitizen` pre-commit hook, which only lints commit messages): `go mod tidy -diff`, `go mod verify`, `go vet ./...`, `go tool staticcheck ./...`, `go test -race -vet=off ./...`. A separate mutating `make tidy` runs `go mod tidy`, `go fix ./...`, `go fmt ./...`. Install `staticcheck` and `sqlc` as tool dependencies in `go.mod` (`go get -tool honnef.co/go/tools/cmd/staticcheck@latest`, `go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest`), not separately managed global binaries — invoke via `go tool staticcheck`/`go tool sqlc`. Prefer this `go get -tool` pattern generally, but check the resulting `go.mod` diff before committing: some CLIs (`golang-migrate/migrate/v4/cmd/migrate`, `pressly/goose/v3/cmd/goose`) unconditionally import every database dialect they support, bloating the module graph by 100+ packages for a Postgres-only project — use the library form of those instead (see Migrations above) rather than their all-dialect CLI.

**Building binaries.** `go build -ldflags='-s' -o=./bin/... ./cmd/...` (strips symbol table, smaller binary); cross-compile explicitly for the homelab's actual OS/arch via `GOOS`/`GOARCH` in addition to any local dev build. `bin/` is gitignored — never commit built binaries. Derive `version` from VCS metadata (`internal/vcs.Version()` via `debug.ReadBuildInfo()`) rather than a hardcoded constant, so it's automatically the Go pseudo-version or exact tag (suffixed `+dirty` on uncommitted changes) — this only populates for `go build`, not `go run`. A `-version` flag prints it and exits immediately after `flag.Parse()`, before any DB/server setup.
