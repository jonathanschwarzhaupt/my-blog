# Coding Standards

This project follows the architectural patterns from *Let's Go* and *Let's Go Further* by Alex Edwards (reference copies: `docs/references/lets-go.html/` and `docs/references/lets-go-further.html/`), adapted for this project's single-binary, two-mode shape (see ADR-0003), templ/templui instead of `html/template`, and Postgres (Neon) instead of MySQL. *Let's Go Further* builds a JSON API (the "Greenlight" movies app) — anything specific to JSON responses is ignored; the operational patterns (connection pooling, migrations, advanced CRUD, pagination, rate limiting, graceful shutdown, metrics, build/release) apply regardless of response format. Where this document contradicts either book, this document wins — the deviations are deliberate and explained below.

Component reference: `docs/references/templui/llms.txt` documents the templui component catalog.

The database layer (sqlc + goose, `internal/database` generated / `internal/models` hand-written) and the `options.go` config pattern follow `jonathanschwarzhaupt/go-cookbook`'s established convention, not the books' hand-rolled model / inline-flags approach.

## Project structure

One binary, two runtime modes, sharing one `internal/` tree:

```
/
├── cmd/
│   ├── blog/            # single binary — public (read-only) or admin (compose/edit), by -features
│   │   ├── main.go
│   │   ├── options.go
│   │   ├── features.go
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

Use a locally-scoped `http.NewServeMux()` in `routes()`. Never rely on `http.DefaultServeMux`.

## Routing

- Go's stdlib mux: method-prefixed patterns (`"GET /{$}"`), wildcards `{slug}`, `{$}` to stop a trailing-slash pattern from acting as a catch-all.
- Most-specific-pattern-wins; keep overlapping patterns to a minimum (ambiguous overlaps panic at startup).
- Validate every wildcard before use, same shape as the book: parse/lookup, and 404 on failure — never let an invalid path value reach a model call.
- Handler naming: `postView` (GET) / `postCreate` (GET, admin) / `postCreatePost` (POST, admin).
- `routes()` returns `http.Handler` (post-middleware), not `*http.ServeMux`.

## Dependency injection

One `application` struct, holding every dependency either mode might need; handlers are methods on it. No globals — except `layout.Features` itself (see Shared-layout feature flags below), which is deliberately the one package-level exception.

```go
// cmd/blog/main.go
type application struct {
    logger  *slog.Logger
    db      database.Querier // sqlc-generated interface (internal/database), see Database section
    baseURL string
    metrics *httpMetrics // always constructed, both modes — see Metrics

    limiter *rateLimiter // only constructed when the admin feature is disabled

    formDecoder    *form.Decoder       // only constructed when the admin feature is enabled
    sessionManager *scs.SessionManager // flash messages only — see Sessions
}
```

`limiter` is dereferenced directly inside `routes()`'s own admin-gated branch. `formDecoder`/`sessionManager` are dereferenced in the admin-only handlers themselves (`post_create.go`, `post_edit.go`, `project_create.go`), not in `routes()` — their nil-safety comes from those handlers only being reachable at all when `routes()` registers the mux entries that call them, which only happens when the admin feature is active.

Never stash long-lived dependencies (DB pool, logger) in request context — only request-scoped data belongs there.

## Configuration

Matches the `options.go` pattern from `jonathanschwarzhaupt/go-cookbook`: one `options` struct + `parseOptions() *options` in `options.go`, keeping `main.go` itself down to wiring, not flag declarations.

```go
// cmd/blog/options.go
type options struct {
    addr           string
    metricsAddr    string // never route this through a public ingress/tunnel — see Metrics
    dbDSN          string
    dbMaxConns     int
    dbMinConns     int
    dbMaxIdleTime  time.Duration
    features       string // comma-separated feature gates, e.g. "admin"
    displayVersion bool
}

func parseOptions() *options {
    opts := &options{}
    flag.StringVar(&opts.addr, "addr", ":4000", "HTTP network address")
    flag.StringVar(&opts.dbDSN, "db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
    flag.StringVar(&opts.features, "features", "", `Comma-separated feature gates to enable, e.g. "admin"`)
    // ...
    flag.Parse()
    return opts
}
```

Use the `flag` package (gives type conversion, defaults, and free `-help`) rather than bare `os.Getenv` for every setting. Pipe environment values (Neon DSN, etc.) in as flag defaults so the same binary works identically whether launched via `make run/...` (env-backed default) or with an explicit override flag — don't make the DSN a hard-required env var with no flag path, since that removes the override needed for tests/local dev pointing at a different database.

`main()` parses `-features` (`features.go`'s `parseFeatures`) into `layout.Features` once at startup, before serving any requests — comma-separated rather than one bespoke flag per feature, deliberately, so it maps directly onto a `features:` array in a future Helm chart being joined into this same flag (see ADR-0003). Unknown names are ignored rather than rejected (startup doesn't fail), mirroring Kubernetes' own tolerance for unrecognized `--feature-gates` entries — but `parseFeatures` still returns them separately so `main()` can log a `Warn`, since a plain typo (`-features=admn`) would otherwise silently deploy an admin instance with no admin routes and no signal anything is wrong.

## Logging

`log/slog`: `slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{...}))`. Structured key-value logging (`logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())`), never `log.Fatal` — log at Error then `os.Exit(1)`. Route the `http.Server`'s own `ErrorLog` through slog via `slog.NewLogLogger(...)`.

**One canonical log line per request**, not several. `logRequest` wraps the response in `statusRecorder` (`middleware.go`) — a small `http.ResponseWriter` wrapper capturing the status code actually sent (defaulting to 200 if `Write` is called without an explicit `WriteHeader`, matching `net/http`'s own default; implements `Unwrap() http.ResponseWriter` for compatibility with `http.ResponseController`) — times the handler, then logs once *after* `next.ServeHTTP` returns with the full picture: `request_id`, `ip`, `proto`, `method`, `uri`, `status`, `duration_ms`. This is a deliberate reordering from logging before serving: a truly hung request now produces no log line until it completes (or is cut off by `ReadTimeout`/`WriteTimeout`), but a single richer line beats a thinner one logged early, and grepping `status=5` across time is now possible at all.

**`request_id`** (`requestid.go`) is a short `math/rand/v2`-generated correlation identifier — not a security token (no `crypto/rand` needed) and not distributed tracing (there's one process and one database, nothing to propagate a trace context to) — assigned by the `requestID` middleware (first in the `standard` chain, before `recoverPanic`, so it's available even if the handler panics), stored via an unexported `contextKey` type, returned as `X-Request-Id`, and included in both the `logRequest` summary line and `serverError`/`render`'s error-path log lines. This is what lets a request's summary line and an error it triggered mid-handler be correlated, even with other requests logged in between.

## Error handling

Centralize in `helpers.go`:

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

Always:
- `standard` chain (wraps the whole mux): `recoverPanic`, `logRequest`, `commonHeaders`, plus `app.limiter.middleware` when the admin feature is *disabled* (skipped entirely in admin mode — Tailscale reachability is already the access control there).

Admin mode only, inside `if layout.Features.Admin` in `routes()`:
- `dynamic` chain (wraps the admin routes only): `preventCSRF`, `sessionManager.LoadAndSave` (see CSRF below).

Public mode has no `dynamic` chain and the admin routes aren't registered at all — not just hidden, genuinely absent from the mux (see Shared-layout feature flags below).

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
| `MaxOpenConns` | `MaxConns` | 25 | 25 | Same reasoning — comfortably below Postgres' default 100-connection hard limit, headroom for both deployment instances sharing one Neon compute. |
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

**Projects (many-to-many with Posts).** `post_projects` is a plain join table (`post_id`, `project_id`, composite PK, both columns `ON DELETE CASCADE`) — no ORM-style association helpers. Projects are never inferred or auto-created from a Post's tags; admin mode only ever assigns a Post to a Project that already exists. Two small helpers in `cmd/blog/project.go` enforce this without a database transaction: `validateProjectIDs` checks every submitted project id against `GetProjectsByIDs` *before* any write, recording a form error if one doesn't exist; `syncPostProjects` then does `DeletePostProjects` + re-`InsertPostProject` per id (replace-all-associations, not a diff) immediately after the Post itself is saved. This isn't wrapped in a transaction with the Post write — see the comment on `syncPostProjects` for why that's an accepted trade-off for a single-admin tool. Foreign-key violations on `post_projects.project_id` (Postgres code `23503`) map to `models.ErrInvalidProject` in `WrapDBError`, alongside the existing unique-violation → `ErrDuplicateSlug` case.

### Migrations (goose)

Single-file migrations under `sql/schema/`, sequentially numbered (`00001_create_posts_table.sql`), each containing `-- +goose Up` / `-- +goose Down` sections — this directory is the single source of schema truth and is committed to version control (not gitignored, unlike `docs/references/`). Conventions: `bigint GENERATED ALWAYS AS IDENTITY` for primary keys, `NOT NULL` + a sensible `DEFAULT` on every column, `text` instead of `varchar(n)`, `CHECK` constraints for business rules, `IF EXISTS`/`IF NOT EXISTS` guards throughout.

`cmd/migrate` is a small standalone binary (goose used as a library, not its all-dialect CLI, to avoid pulling in every driver goose/golang-migrate support — MySQL, Cassandra, Vertica, etc. — for a Postgres-only project) that embeds `sql/schema` via `embed.FS` and runs `goose.Up`/`Down`/`Status` against a DSN flag. It's invoked locally via `make db/migrations/up`, and is built as its own minimal container image to run as a **Kubernetes init container** ahead of both `blog` deployments in the homelab — migrations are never run automatically from the server binary's own startup path.

## Filtering, sorting, pagination

Aspirational — not implemented yet. Public mode's home page, feed, Project page, and Projects index all currently fetch and render every row unpaginated (a deliberate, accepted trade-off given the blog's small scale); apply the pattern below once post/project volume actually warrants it, to any listing endpoint (home page / Project page in public mode, the post list in admin mode):

- Shared `Filters` struct (`Page`, `PageSize`, `Sort`, `SortSafelist []string`), validated: `Page` capped well below any realistic post count, `PageSize` capped at 100, `Sort` checked against `SortSafelist` via `validator.PermittedValue`.
- **Never interpolate raw sort input into SQL.** The only place `fmt.Sprintf` is acceptable for building a query is injecting a column/direction that has already been checked against `SortSafelist` — placeholders can't parameterize identifiers.
- Always add a secondary `ORDER BY ..., id ASC` — Postgres doesn't guarantee stable order without a unique tiebreaker, which matters once pagination is involved.
- Pagination via `LIMIT`/`OFFSET`; get the total count in the same query via a window function (`SELECT count(*) OVER(), ... LIMIT $n OFFSET $m`) rather than a separate `COUNT(*)` query.

## Forms & validation

`internal/validator` package, embedded in every form struct: `CheckField`, `AddFieldError`, `AddNonFieldError`, `Valid()`, plus `NotBlank`, `MaxChars`, `MinChars`, `PermittedValue[T comparable]`.

Decode POST bodies with `go-playground/form` via `app.decodePostForm(r, &form)` rather than manual `r.PostForm.Get(...)` parsing.

On validation failure: re-render the same page with **422 Unprocessable Entity**, passing the form struct (values + `FieldErrors`) as a typed parameter into the templ component — same idea as the book's `templateData.Form any` field, but as an explicit typed argument to the component function rather than a template action.

## CSRF (admin mode only)

Admin mode's state-changing forms (compose, edit) must be protected even though that deployment is tailnet-only — Tailscale prevents *unauthorized network access*, not a malicious page in your browser submitting a forged POST while you're on the tailnet.

Deliberate deviation from the book: use the modern stdlib approach instead of adding `justinas/nosurf` as a dependency —

- `scs` already sets `SameSite=Lax` on the session cookie by default — keep it.
- Add `http.CrossOriginProtection` middleware (Go stdlib) to the `dynamic` chain, checking `Sec-Fetch-Site`/`Origin`.
- No CSRF token field needed in forms with this approach (unlike `nosurf`'s hidden `csrf_token` input).

This is simpler and dependency-free, appropriate for a single-user admin tool that doesn't need to support pre-2020 browsers.

## Sessions (admin mode only, flash messages only)

Per the network-boundary decision (ADR-0001, superseded on the binary split but not on this point — see ADR-0003), admin mode has **no login system** — reaching it over Tailscale is the authentication. `scs` sessions exist solely for flash messages (e.g. "Post published"), not identity:

```go
sessionManager := scs.New()
sessionManager.Store = <driver>store.New(db)
sessionManager.Lifetime = 12 * time.Hour
```

`Put(ctx, "flash", msg)` on write, `PopString(ctx, "flash")` in a `newTemplateData`-equivalent helper so every render surfaces and clears it. No `RenewToken`, no `authenticatedUserID`, no `requireAuthentication` middleware — there is no authenticated-vs-anonymous distinction to make.

Public mode has no sessions at all — `sessionManager` stays `nil` there.

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

**Gotcha:** a raw text node whose first word is `for`, `if`, or `switch` right after a tag closes (e.g. `</a> for evidence...`) makes `templ generate` misparse it as the start of a control-flow block instead of plain text. There is a real fix — wrap just that word in a Go string expression, e.g. `{ "for" }` — but the error message you get depends on nesting: at the top level of a `templ` block it names the actual problem (`for: unterminated ... to escape "for", "if", "switch" etc. with braces, e.g. '{ "for" }'`); nested inside another component's block argument (e.g. anything wrapped in `@layout.Base(...) { ... }`, which is effectively every real page) it instead surfaces as a much less obvious `expected nodes, but none were found` pointing at an unrelated line — that's the form you'll likely hit in practice.

### templui + Tailwind CSS

Styling is [templui](https://templui.io) (a templ component library) on top of Tailwind CSS v4, both self-hosted — no CDN, no runtime JS framework beyond what a given templui component itself needs.

- **templui's CLI is a `go get -tool`** (`github.com/templui/templui/cmd/templui`), invoked as `go tool templui ...`, consistent with sqlc/templ/staticcheck. `templui init` (already run) wrote `.templui.json`, pointing `componentsDir`/`utilsDir` at `ui/templ/components`, `jsDir` at `ui/static/js`, `jsPublicPath` at `/static/js` — matching this project's existing `ui/templ`/`ui/static` layout rather than templui's own defaults (`components`/`assets/js`).
- **`templui add <component>...`** copies a component's `.templ` source (and any JS it needs) directly into `ui/templ/components/` — committed, owned source, not a live dependency. **Unlike `internal/database` (sqlc-generated, never hand-edit, changes only go through `sql/queries/*.sql` + regenerate), templui components are meant to be hand-edited** — that's templui's whole "customize everything, own your code" model (shadcn-style), the opposite convention from sqlc. The only thing to watch for: re-running `templui add <component>` (or `--installed` to update everything) overwrites that file from templui's registry, silently discarding any local edits — treat that command as a deliberate, occasional "take the upstream version instead of mine" action, not something to run routinely. Only add components a page actually uses; don't bulk-install the whole catalog.
- **Tailwind CSS is the standalone CLI binary**, managed via `mise` (`aqua:tailwindlabs/tailwindcss` in `mise.toml`) rather than npm — no `package.json`/`node_modules` needed even though Node is already available in this project's toolchain for other reasons. Invoked in the Makefile as `mise exec -- tailwindcss ...` rather than bare `tailwindcss`, since mise's per-project tool shims aren't guaranteed to be on `PATH` in every shell that might run `make` (a CI runner, a deploy script) the way they are in an interactive dev shell with `mise activate` sourced. `ui/css/input.css` is the source (Tailwind config lives in CSS itself in v4, not a JS config file, and its `@source "../templ"` directive is relative to `ui/css/`'s own location — if `input.css` ever moves, that path needs updating too, since a wrong `@source` fails silently by just omitting the missed utility classes, not with a build error); `make css/build` compiles it to `ui/static/css/main.css`, which is what's actually embedded via `//go:embed` in `ui/embed.go` — `ui/css/` itself is not embedded, it's build-time-only input. `run/blog`, `run/blog-admin`, and `build/blog` Makefile targets all depend on `css/build`, so the compiled CSS is never stale for local dev; `make audit` additionally fails loudly if rebuilding actually changes `ui/static/css/main.css` from what was on disk, catching a stale committed CSS file before it ships.
- **Personality theme** (see ADR-0005), set once as CSS custom properties in `ui/css/input.css`'s `:root` block — warm paper background/ink (`#F4EFEA`/`#1d1f27`), a red-orange primary accent (`#f54e00`) and a mustard secondary accent (`--accent-secondary`, `#f7a501`), all sourced from PostHog's and MotherDuck's actual compiled CSS rather than a generic hue pick. Still no dark mode or theme switching shipped — but every value stays a semantic custom property, so a future dark variant is a values-only addition in that same block, not a `.templ` rewrite. A "boxy detail" layer sits on top of templui's unchanged shape (0.65rem radius, 1px borders stay put): nav links, tag/badge pills, the footer quip, and post titles render in a self-hosted JetBrains Mono (`--font-mono`, OFL-licensed, `ui/static/fonts/`) rather than the body's system-sans stack, and the footer's top border is a double rule instead of a hairline. Body copy is deliberately excluded from the monospace treatment to keep daily-writing prose readable.
- **`ui/templ/components/utils/templui.go`** (copied by `templui init`) provides small helpers like `utils.TwMerge` for conflict-resolving combined Tailwind classes — use it when a component's classes are built up conditionally rather than hand-rolling string concatenation.

### Post body markdown (`internal/markdown`)

Post `Body` is stored and edited as plain markdown source — no schema change, no change to the compose/edit forms. Rendering to HTML happens at **display time only**, in `postView` (`cmd/blog/post_view.go`), via `internal/markdown.Render` (wraps [goldmark](https://github.com/yuin/goldmark) with the GFM extension bundle — tables, strikethrough, autolinks, task lists; fenced code blocks are core CommonMark, no extension needed). The rendered HTML string is passed into `PostView` as a second parameter and embedded with `@templ.Raw(bodyHTML)` — `templ.Raw` is templ's own runtime type, already imported into every generated file, so don't add an explicit `"github.com/a-h/templ"` import to the `.templ` source itself (it'll collide: `templ redeclared in this block`).

**Raw HTML in post source is dropped, not escaped or rendered** — goldmark's default behavior with `html.WithUnsafe()` left off (deliberately not enabled): both block-level and inline raw HTML are replaced with an `<!-- raw HTML omitted -->` comment. This is a single-trusted-author blog, so passthrough wouldn't really be "unsafe" in the usual multi-user sense, but there's no concrete need for it in post content, and leaving it off closes the question before it's ever asked.

Rendering errors from `markdown.Render` map to `serverError`, same as any other handler failure — not a silent fallback to the raw markdown text. In practice this is rare: CommonMark parsers render best-effort on malformed syntax rather than failing, so `Convert` only errors on genuine writer-level failures.

Not yet done, deliberately out of scope for now: syntax highlighting for code blocks (`goldmark-highlighting` + `chroma` is a meaningfully heavier dependency than goldmark alone), a live preview in the compose/edit form, and markdown rendering for `SoWhat` or Project descriptions (both stay plain text).

### Shared-layout feature flags (`ui/templ/layout/features.go`)

`layout.Features` (a package-level `FeatureFlags` struct, currently just `Admin bool`) gates both which nav sections `base.templ` renders *and* which routes `routes()` registers (see ADR-0003) — the one deliberate package-level global in the codebase (see Dependency injection above). `main()` sets it once at startup from the parsed `-features` flag, before serving any requests, and it's never mutated afterward. Because the public routes (Home, Projects, About, feed) are always registered regardless of mode, admin mode's nav links to them are real links, not dead ones — unlike the old two-binary arrangement, where `blog-admin` rendered those links without actually serving them.

## Client-side interactivity (Alpine.js / Alpine AJAX)

Default is plain server-rendered HTML: standard `<form>` posts, full-page navigations. No JS is added by default.

- **Alpine.js**: only for small, local, ephemeral UI state that doesn't need the server at all — a mobile nav toggle, a client-side character counter on the So What field. Written inline via `x-data` in the templ markup, no build step.
- **Alpine AJAX**: only when a specific interaction genuinely benefits from a partial-page update (e.g. filtering posts by tag/Project without a full reload) — and only *after* the plain-HTML, full-reload version of that same endpoint already works. Alpine AJAX enhances the existing route/fragment; it never replaces the no-JS path.
- Vendor both into `ui/static/` (embedded, same as CSS) rather than pulling from a CDN — consistent with self-hosting everything else.

If a feature doesn't need either, don't add either.

## Server config & timeouts

Construct `*http.Server` explicitly instead of `http.ListenAndServe`, with `IdleTimeout`, `ReadTimeout`, `WriteTimeout` always set explicitly (mitigates Slowloris-style slow-client issues; `IdleTimeout` doesn't default from `ReadTimeout`).

Deliberate deviation from the book: the binary never terminates TLS itself, in either mode. Public mode's TLS is terminated at Cloudflare Tunnel; admin mode's transport security comes from the Tailscale (WireGuard) network layer. Both deployments serve plain HTTP locally. The book's self-signed-cert/TLS-config chapter (09.03–09.05) doesn't apply here.

**Graceful shutdown.** The binary (running as a long-lived homelab service, in either mode) catches `SIGINT`/`SIGTERM` on a buffered `chan os.Signal, 1`, then calls `srv.Shutdown(ctx)` with a bounded context (~30s) and exits cleanly rather than dropping in-flight requests. `http.ErrServerClosed` from `ListenAndServe` is the expected/good outcome, not an error to log. Server construction lives in its own `server.go`/`serve()` method, not inline in `main()`.

Two `*http.Server`s actually run per process: the main app server (`serve()`) and the metrics server (`serveMetrics()`, see Metrics below), both on the same `SIGINT`/`SIGTERM`-derived context. `main()` waits on both before exiting (a `metricsDone` channel), not just the main server — letting the metrics goroutine get abandoned mid-shutdown when `main()` returns would undo its own graceful shutdown, since Go terminates all goroutines the instant `main()` returns regardless of what they were doing.

## Rate limiting (public mode only)

Public mode is internet-facing and should defend against scraping/abuse; admin mode is tailnet-only, so the limiter is skipped there entirely rather than just relaxed — Tailscale reachability is already the access control, and a rate limiter there would only risk throttling legitimate use for no real security benefit (see `routes()`'s `layout.Features.Admin` check).

- Per-client token bucket via `golang.org/x/time/rate`: a `map[string]*client{limiter, lastSeen}` keyed by IP, guarded by a `sync.Mutex` (unlocked explicitly before calling `next.ServeHTTP`, not deferred).
- A background goroutine sweeps the map every minute, evicting entries older than a few minutes, to bound memory.
- Resolve the real client IP via a real-IP helper that checks `Cf-Connecting-Ip` first — Cloudflare's edge sets this and it cannot be spoofed by the client, unlike `X-Real-IP`/`X-Forwarded-For`, which any client can set to an arbitrary value and are only safe to trust as fallbacks for non-Cloudflare contexts (e.g. local dev behind a different reverse proxy) — before finally falling back to `r.RemoteAddr`.
- Configurable via flags (`rps`, `burst`, `enabled`) so it can be disabled for local dev/load testing without a code change.
- Note this in-memory approach only works for a single instance — fine here since there's exactly one public-mode process, but wouldn't survive a move to multiple replicas without an external store.

## RSS feed (public mode primarily, but always registered)

`GET /feed.xml` generates an RSS 2.0 document from the same `ListPosts` query backing the home page (same order, newest-first) — built with `encoding/xml` (`rssFeed`/`rssChannel`/`rssItem` in `cmd/blog/feed.go`), not a templ component, since it's XML rather than HTML. Each item's `link`/`guid` is `app.baseURL + "/posts/" + slug`, so absolute link correctness depends entirely on `-base-url` being set correctly in production — `main.go` logs a `Warn` at startup if it's left at the `http://localhost:4000` default, since a misconfigured value silently produces unusable feed links with no runtime error. `Description` is each post's So What, not a body excerpt.

## Metrics (ops, not page analytics)

Separate from the self-hosted Umami/Plausible *page-view* analytics (Q13 of the design) — these are internal operational metrics for kube-prometheus-stack/Grafana dashboards in the homelab, not visitor tracking.

**`github.com/prometheus/client_golang`, not `expvar`.** An earlier version of this doc described an `expvar`-based design, following *Let's Go Further*'s own approach — but `expvar` emits a bespoke JSON format that Prometheus cannot scrape. `promhttp.Handler()` (Prometheus's actual wire format) is the correct tool whenever the consumer is a real Prometheus server, which is the case here.

**Exposed on a separate port (`-metrics-addr`, default `:9091`), not the app's main port.** A second `*http.Server` (`serveMetrics` in `server.go`, mirroring `serve()`'s exact graceful-shutdown shape) serves only `promhttp.HandlerFor(registry, ...)` — completely separate from `routes()` and the app's mux. **This port must never be referenced in the Cloudflare Tunnel's ingress config** — that's what keeps it unreachable from the public internet. kube-prometheus-stack's Prometheus scrapes it in-cluster via a `PodMonitor`/`ServiceMonitor` pointed directly at the port over the cluster network, never touching the tunnel. This is the same network-boundary-over-application-auth principle already used for admin routes (Tailscale) — access control here is a Kubernetes networking concern, not something the endpoint itself checks. **No authentication is added to `/metrics`**; that's deliberate, not an oversight.

Metrics are constructed and exposed in **both modes** (public and `-features=admin`), unconditionally — `application.metrics` (`*httpMetrics`) is always non-nil, since this is operational instrumentation orthogonal to the admin/public route-gating `layout.Features` controls.

**v1 metric set, deliberately unlabeled by route** to sidestep cardinality risk (labeling by exact path would mean injecting a safe pattern label like `/posts/{slug}` at every route registration in `routes.go` — not done, only worth it if per-route breakdown becomes a genuine need):

- **HTTP** (`httpmetrics.go`): `blog_http_requests_total` and `blog_http_request_duration_seconds`, both labeled by `method` and `status` only. The middleware reuses `statusRecorder` (the same wrapper `logRequest` uses — see Logging above) rather than reinventing status capture; it's a separate wrap in its own middleware, since logging and metrics are separate concerns even though both need the status code.
- **Go runtime** (`metrics.go`): `collectors.NewGoCollector()` and `NewProcessCollector()` — goroutines, GC, memory, CPU, open file descriptors, all for free, no custom code.
- **DB pool** (`dbpoolcollector.go`): a custom `prometheus.Collector` (`Describe`/`Collect`, not a fixed gauge set updated on a timer) reading `pool.Stat()` fresh on every scrape — `blog_db_pool_max_conns`, `_acquired_conns`, `_idle_conns`, `_total_conns`, `_acquire_count_total`.
- **Build info** (`metrics.go`): `blog_build_info{version="..."} 1`, the standard "build_info" pattern most Prometheus-instrumented Go services use, sourced from `internal/vcs.Version()`.

Kubernetes `PodMonitor`/`ServiceMonitor` manifests, Grafana dashboard JSON, and alerting rules live in the homelab's own infrastructure/GitOps repo (or a future Helm chart for this blog), not here — this project's job stops at correctly exposing the metrics.

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
- Admin-mode end-to-end tests: a `testServer` wrapping `httptest.NewServer(app.routes())` (no TLS needed locally per the deviation above) with `get()`/`postForm()` helpers and a cookie jar for session persistence; reset the jar between sub-tests. Since `routes()` is now shared across both modes, tests set `layout.Features.Admin` explicitly before building the server rather than relying on which binary the test file happens to live in.
- Mock the database dependency by implementing sqlc's generated `database.Querier` interface with fixture data — no hand-written interface to maintain, since `emit_interface: true` generates it.
- Integration tests against a real test Postgres (Neon branch or local instance): `newTestDB(t)` running goose migrations from `sql/schema` against a scratch database, `t.Cleanup` tearing it down; skip via `testing.Short()`.

## Build & release

**Makefile** at the repo root, targets namespaced with `/` (`db/migrations/up`, `run/blog`, `run/blog-admin`, `build/blog`), never `:` in target names. All action-only rules marked `.PHONY`. A `confirm` prerequisite guards destructive targets (e.g. running migrations against a real DB): `@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]`. A self-documenting `help` target (parses `## target: description` comments from the Makefile itself) is the default (first) rule, so bare `make` prints usage.

**Environment variables.** No secrets or DSNs hardcoded in `main.go` — flags default to `""`, and a gitignored `.envrc` (added to `.gitignore` the moment it's created) supplies real values via `include .envrc` in the Makefile, injected as `${VAR}` into `make run/...` targets. Flags remain the only way the running binary is actually configured; env vars are just a dev-time convenience for populating them. **Don't quote values in `.envrc`** (`export VAR=value`, not `export VAR="value"`) — Make's `include` parses `export NAME = value` as native syntax rather than sourcing it through a shell, so quote characters become part of the value literally and corrupt anything that reads the env var directly (e.g. a Go test calling `os.Getenv`) rather than through a `${VAR}`-interpolated recipe line, where the shell strips them.

**Quality control**, run via `make audit` before committing (distinct from the existing `commitizen` pre-commit hook, which only lints commit messages): `go mod tidy -diff`, `go mod verify`, `go vet ./...`, `go tool staticcheck ./...`, `go test -race -vet=off ./...`. A separate mutating `make tidy` runs `go mod tidy`, `go fix ./...`, `go fmt ./...`. Install `staticcheck` and `sqlc` as tool dependencies in `go.mod` (`go get -tool honnef.co/go/tools/cmd/staticcheck@latest`, `go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@latest`), not separately managed global binaries — invoke via `go tool staticcheck`/`go tool sqlc`. Prefer this `go get -tool` pattern generally, but check the resulting `go.mod` diff before committing: some CLIs (`golang-migrate/migrate/v4/cmd/migrate`, `pressly/goose/v3/cmd/goose`) unconditionally import every database dialect they support, bloating the module graph by 100+ packages for a Postgres-only project — use the library form of those instead (see Migrations above) rather than their all-dialect CLI.

**Building binaries.** `go build -ldflags='-s' -o=./bin/... ./cmd/...` (strips symbol table, smaller binary); cross-compile explicitly for the homelab's actual OS/arch via `GOOS`/`GOARCH` in addition to any local dev build. `bin/` is gitignored — never commit built binaries. Derive `version` from VCS metadata (`internal/vcs.Version()` via `debug.ReadBuildInfo()`) rather than a hardcoded constant, so it's automatically the Go pseudo-version or exact tag (suffixed `+dirty` on uncommitted changes) — this only populates for `go build`, not `go run`. A `-version` flag prints it and exits immediately after `flag.Parse()`, before any DB/server setup.

**Hot reload (air)**, via `make dev/blog` / `make dev/blog-admin` — one binary, but each mode still gets its own `.air.blog.toml` / `.air.blog-admin.toml`, since air's `full_bin`/`[proxy]` config is per run-configuration (different flags, different port), not per Go package.

- **Installed via `mise`** (`air = "latest"` in `mise.toml`), not `go get -tool` — `go get -tool github.com/air-verse/air@latest` was tried first and rejected: it pulls in the entire Hugo static site generator (SASS/SCSS compilers included) as a transitive dependency, 14 new indirect `go.mod` entries for a tool that never ships in the binary itself. Same class of problem as the golang-migrate/goose CLI bloat documented above — `mise` (already the mechanism for `tailwindcss`) keeps genuinely dev-only tools out of the module graph entirely.
- Both configs' `[build].cmd` is `make build/blog` — air's own docs explicitly endorse `cmd = "make ..."` — so hot reload goes through the exact same `templ generate` → `css/build` → `go build` chain as a real build, not a separate parallel path that could drift. Only `full_bin` differs between the two configs (`-features=admin` or not).
- `exclude_regex = ["_templ\\.go$", "_test\\.go$"]` is load-bearing, not cosmetic: `cmd`'s own `templ generate` step rewrites `_templ.go` files on every rebuild, and without excluding them from the watch, that rewrite would immediately retrigger another rebuild — an infinite loop.
- `full_bin` (e.g. `./bin/blog -addr=:8080`) doesn't pass `-db-dsn` explicitly and doesn't go through the Makefile's own `${BLOG_DB_DSN}` variable interpolation (that's Make-internal, not an OS environment export) — so `.envrc` must already be sourced in the shell before `make dev/blog`, same requirement as running `go run ./cmd/blog` directly without the `-db-dsn` flag.
- `[proxy]` is enabled (browser auto-refreshes after each rebuild) on a separate port per mode (`8091`/`4091`) so both can run simultaneously without colliding.

## Release pipeline

Background research and the primary-source citations behind the decisions below live in `docs/research/release-pipeline.md` (release-please/GoReleaser/Docker tagging) and `docs/research/neon-branching.md` (Neon branch mechanics and CI patterns) — this section records the decisions actually taken, not the research itself. `docs/neon-branches.md` documents the Neon branch topology itself in more detail.

**Scope boundary.** This repo's responsibility ends at a correctly-tagged, correctly-versioned image landing in GHCR (or, for the dev channel, a floating tag) and a database branch existing with its DSN recorded somewhere consumable. The Kubernetes manifests, GitOps policies (Flux `ImagePolicy` etc.), and Renovate config that actually deploy those images live in the separate homelab infrastructure repo, not here.

### Three-branch model

| Branch | Role | Merge strategy in | Why |
|---|---|---|---|
| `feature/*` | One per issue/PRD chain, branched off `development` | Squash-merged into `development` | A feature branch typically accumulates messy in-progress commits (fixups, review-response commits); squashing collapses that into one clean commit before it ever reaches a branch release-please watches. |
| `development` | Persistent integration branch; every push triggers the dev-image workflow | Regular merge commit into `main` (never squashed) | release-please derives one changelog entry per conventional-commit-formatted commit on the branch it watches. Read directly from release-please's own source (`src/commit.ts`'s `splitMessages()`), GitHub's default squash-merge commit body (`* ` bullets, single newlines) does not get split back into multiple entries by that parser, so squashing `development → main` would collapse an entire batch of features into a single undifferentiated changelog line. A regular merge commit preserves each feature's own commit, so `main` gets one changelog entry per feature — the reason this level's strategy is deliberately the opposite of the level below it. |
| `main` | release-please's target; a merge here is a real release event | — | Tags and GitHub Releases are only ever cut from here. |

### release-please configuration

`release-please-config.json` uses a single package at the repo root (`"."`), not `linked-versions` or a multi-package layout — there is exactly one version governing both `cmd/blog` and `cmd/migrate` binaries, and per the manifest engine's own design a single-entry manifest is the recommended shape for a single-artifact repo, not a special/reduced mode. `release-type: "go"` (no `version-file`, since nothing in this repo needs a version constant rewritten — version comes from VCS metadata at build time, see Building binaries above). `include-component-in-tag: false` + `include-v-in-tag: true` produce a plain `vX.Y.Z` tag rather than a component-prefixed one, since there's only one component. `initial-version: "0.1.0"`, since the repo had zero prior tags when this was configured.

**The default `GITHUB_TOKEN` is not sufficient for the release-please job itself** — discovered empirically, not predicted by the design: GitHub's own recursive-workflow guard means a `GITHUB_TOKEN`-authored push/PR does not trigger further workflow runs, so `RELEASE_PLEASE_TOKEN` (a PAT) is required for `release.yml`'s `release-please` job to create the Release PR and, later, the tag push that other workflows might key off. The `build-and-push` job triggered by `release_created` in that same workflow, by contrast, only reads `secrets.GITHUB_TOKEN` for GHCR auth (`packages: write` is enough) — the PAT requirement is specific to release-please's own PR/tag-creation step, not to publishing images.

### Why GoReleaser is not used

GoReleaser was deliberately dropped from the design, not omitted by oversight. Two independent reasons:

- **Version injection mechanism mismatch.** `internal/vcs.Version()` (see Building binaries above) reads Go's own VCS build-info stamping via `debug.ReadBuildInfo()` — it needs `.git` present in the build context and real commit history, and is populated automatically by `go build` alone. GoReleaser's convention is the opposite: inject the version via `-ldflags` into a package-level variable. Adopting GoReleaser would mean maintaining two competing version mechanisms or ripping out the one already in place for no benefit.
- **No floating dev-channel mechanism in OSS GoReleaser.** `--snapshot` deliberately never publishes anything (`--skip=announce,publish,validate` is implied). `--nightly` — the actual rolling/floating-tag feature — is GoReleaser Pro-only. Since a free/OSS pipeline for the dev-image channel would need a separate `docker/build-push-action`-based step regardless, routing the tagged-release channel through GoReleaser and the dev channel around it would mean two different build mechanisms for the same Dockerfiles — so both channels use the same plain `docker/build-push-action` mechanism instead, and GoReleaser is not used anywhere in the pipeline.

### Docker images

Both binaries are built as separate multi-stage images (`Dockerfile.blog`, `Dockerfile.migrate`): a `golang:1.26` build stage running natively on the build platform (`--platform=$BUILDPLATFORM`) cross-compiling via `GOOS`/`GOARCH` (no QEMU emulation), producing `linux/amd64` + `linux/arm64` images from one build; a `gcr.io/distroless/static-debian12` final stage (no shell, minimal attack surface, includes the CA cert bundle Neon's `sslmode=require` needs). `.dockerignore` excludes `docs/`, `bin/`, `.devcontainer/` — but never `.git`, since the build stage's VCS version stamping depends on it being present. Both images publish to GHCR (`ghcr.io/jonathanschwarzhaupt/home-blog`, `ghcr.io/jonathanschwarzhaupt/home-blog-migrate`) via the default `GITHUB_TOKEN` (`packages: write`), no PAT needed for this specific purpose.

Two channels, two tag schemes:

- **Tagged release** (`release.yml`, triggered on release-please cutting a real tag on `main`): `vX.Y.Z` and a floating `:latest`.
- **Dev channel** (`dev-image.yml`, triggered on every push to `development`): computes the next version via `release-please release-pr --dry-run` against `development`'s HEAD — read-only against GitHub's API, creates no PR or tag — and tags the image `X.Y.Z-dev.<run_number>.<short-sha>` (a valid SemVer 2.0 prerelease identifier, and a valid Docker tag, since Docker tags can't contain `+`) plus a floating `:dev`. This never touches `main`'s tag list or release-please's own bookkeeping there.

### Neon branch topology

Three distinct branches of the same Neon project, not three different databases: `production` (the one real database, fed only on a tagged release — see ADR-0002/ADR-0003 for which deployment that is), a persistent `development` branch isolated from it (feeding the always-on admin-preview environment the homelab repo runs), and ephemeral `preview/pr-<number>` branches (created by `migration-check.yml` per open PR touching migration-relevant paths, deleted on success, left alive on failure for debugging). `docs/neon-branches.md` is the source of truth for this topology and where each branch's DSN lives — this section only records the two facts specific to the CI pipeline itself:

- All Neon DSNs used in CI are direct/unpooled connections, never Neon's `-pooler` endpoint — PgBouncer's transaction-mode pooling doesn't preserve session state across statements, which breaks goose's session-level advisory lock (a different, narrower reason than `internal/models`' long-running-pool rationale above — see Database layer).
- GitHub Environments (not plain repo secrets, which have no per-workflow scoping) hold the `NEON_API_KEY`/`NEON_PROJECT_ID` used to provision the ephemeral branches.
