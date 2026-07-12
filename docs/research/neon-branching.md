# Research: Neon Postgres branching + `neonctl` for CI/CD

Scope: primary-source findings on Neon's branching model, the `neonctl` CLI, and Neon's own
GitHub Actions, for a design discussion about ephemeral per-dev-build/per-PR databases in this
repo (`cmd/blog` + `cmd/migrate`, already running on Neon in production — see
`CODING_STANDARDS.md`'s "Database layer" section and `.envrc`). This is a companion to
`docs/research/release-pipeline.md` (release-please + GoReleaser design); it does not repeat
that doc's content, only its vocabulary (the still-unresolved dev-channel Docker-image question
from that doc is referenced in Open Questions below).

Research method: primary sources only — `neon.com/docs/*` (Neon's docs migrated from
`neon.tech` to `neon.com`; all `neon.tech/docs/...` URLs 308-redirect there) and
`github.com/neondatabase/*` repos, fetched directly (`WebFetch`/`gh api`), July 2026. Where a
search engine's own synthesis is quoted below, it was cross-checked against the linked primary
page.

---

## 1. What a Neon branch actually is

**Copy-on-write, not a copy.** Per Neon's architecture overview
(https://neon.com/docs/introduction/architecture-overview): "When you create a branch in Neon,
the engine does not duplicate files or pages. Instead, the new branch points to an existing
point in history and begins diverging from there using copy-on-write semantics." Branch
creation is a metadata operation against the storage layer's immutable history, not a bulk data
copy — new storage is only consumed once the branch's data actually diverges from its parent.

**Compute/storage split is what makes this possible.** Same page: "Instead of running Postgres
as a single stateful system tied to a VM and its filesystem, Neon is a serverless database that
splits the system into two independent layers: compute and storage." Compute (the Postgres
process itself) is ephemeral/stateless; storage (safekeepers + pageserver + object storage)
holds all durable state and branch history independently of any particular compute.

**Branch = a copy-on-write clone of data, not compute.** Per
https://neon.com/docs/introduction/branching: "A branch is a copy-on-write clone of your data,"
and "Writes to a branch are saved as a delta" — changes are tracked incrementally per-branch,
not by duplicating the parent's full dataset. "Creating a branch does not increase load on the
parent branch or affect it in any way" — branch creation is documented as zero-impact on the
parent.

**Each branch gets its own compute.** The same page notes tests "can also run on separate
branches in parallel, with each branch having dedicated compute resources" — i.e., a branch is
a storage-layer concept, but using it (connecting, running queries/migrations) requires an
associated compute endpoint, which is a separate resource attached to the branch (see §5, §6).

**Branches can be created from a point in time, not just "now."** `neonctl branches create
--parent` accepts "parent branch name, ID, timestamp, or LSN" (§2) — branching isn't only a
snapshot of the current tip, it can also instantiate history from an earlier point (used for
point-in-time restore, not just dev/test branch-off).

---

## 2. `neonctl` CLI: branch subcommands

Source: https://neon.com/docs/reference/cli-branches (fetched directly; `neon.tech/docs/...`
redirects here per Neon's `neon.tech` → `neon.com` domain move).

Subcommands on `branches`: `add-compute`, `create`, `delete`, `get`, `list`, `rename`, `reset`,
`restore`, `schema-diff`, `set-default`, `set-expiration`.

**`neon branches create`**
```
neon branches create [options]
```
Key flags: `--name` (branch name, up to 256 chars, must be unique), `--parent` (parent branch
name, ID, **timestamp, or LSN** — defaults to the project's default branch), `--compute`
(create with compute; default `true`, `--no-compute` to omit), `--type` (`read_write` default,
or `read_only` for a read replica), `--cu` (Compute Units, fixed e.g. `"2"` or a range e.g.
`"0.5-3"`), `--expires-at` (RFC 3339 expiration timestamp), `--schema-only`, `--psql` (open a
psql session immediately), `--project-id`.

Example (point-in-time / "instant restore" branch): `neon branches create --name data_recovery
--parent 2023-07-11T10:00:00Z`.

**`neon branches delete <id|name>`** — deletes a branch by ID or name, e.g. `neon branches
delete br-rough-sky-158193 --project-id crimson-voice-12345678`. Key flag: `--project-id`.

**`neon branches reset <id|name>`** — per the doc: "Resets a child branch to the latest data
from its parent." The `--parent` flag is **required**, and per the doc text, "resetting from
the parent branch is currently the only supported reset operation" — i.e. `reset` as of this
writing is *only* "reset child to parent's current tip," not "reset to an arbitrary
LSN/timestamp." (Point-in-time branch creation, §1 above, is the mechanism for the
timestamp/LSN case — it creates a *new* branch at that point, it doesn't reset an existing one
to it.) Other flags: `--preserve-under-name` (keeps the pre-reset branch under a given name
instead of discarding it), `--project-id`.

**`neon branches list [--project-id ...]`** — lists branches in a project.

**`neon connection-string [branch] [options]`** (https://neon.com/docs/reference/cli-branches,
cross-referenced with the `connection-string`/`cs` command; the `reset-branch-action` source in
§3 confirms the short alias `neonctl cs`): prints a ready-to-use connection URI, e.g.
`postgresql://alex:AbC123dEf@ep-cool-darkness-123456.us-east-2.aws.neon.tech/dbname?sslmode=require&channel_binding=require`.
Flags: positional `[branch]` (name/ID; omit for default branch; `branch@timestamp` or
`branch@lsn` selects a point-in-time connection), `--database-name`, `--role-name`,
`--project-id`, `--pooled` (adds the `-pooler` suffix, i.e. routes through Neon's PgBouncer
pooler endpoint rather than the direct/unpooled endpoint — see §5's relevance to this repo's
`pgxpool`-direct-connection convention), `--endpoint-type read_only`, `--prisma` (adds
`connect_timeout=30`), `--ssl` (`require`/`verify-ca`/`verify-full`/`omit`), `--psql`,
`--extended` (adds extra fields like `host`/`password` to JSON output — used by
`reset-branch-action`, see §3), `-o/--output json`.

---

## 3. Neon's own GitHub Actions — exact inputs/outputs, read from the actual `action.yml`/`action.yaml`

All three read directly from the repos' own action manifest files via `gh api
repos/neondatabase/<repo>/contents/action.yml(.yaml)`, not README paraphrase.

### `neondatabase/create-branch-action` (https://github.com/neondatabase/create-branch-action)
`runs.using: node24` (compiled JS action, calls the Neon API directly — not a `neonctl` wrapper).
Inputs: `api_key` (required), `api_host` (default `https://console.neon.tech/api/v2`),
`branch_name`, `project_id` (required), `parent_branch`, `prisma` (default `false`), `database`
(default `neondb`), `role` (default `neondb_owner`), `branch_type` (`default`/`schema-only`),
`ssl` (default `require`), `suspend_timeout` (seconds of inactivity before the new branch's
compute suspends; default `0`), `expires_at`, `masking_rules`, `get_auth_url`,
`get_data_api_url`.
Outputs: **`db_url`, `db_url_pooled`, `db_host`, `db_host_pooled`, `password`, `branch_id`,
`created`** (bool — branch newly created vs. an existing branch of that name was reused),
`auth_url`, `data_api_url`. **Crucially, `db_url`/`db_url_pooled` are full ready-to-use
Postgres connection strings emitted directly as step outputs** — no separate query step is
needed to get a usable DSN out of branch creation.

### `neondatabase/delete-branch-action` (https://github.com/neondatabase/delete-branch-action)
`runs.using: composite` — a thin shell wrapper: `npm i -g neonctl@2.22.0`, then literally
`neonctl branches delete "$INPUT_BRANCH" --project-id "$INPUT_PROJECT_ID"` (or `$INPUT_BRANCH_ID`
if `branch` isn't set — `branch_id` is marked deprecated in favor of `branch`). Inputs:
`project_id` (required), `branch` (name or ID), `branch_id` (deprecated), `api_key` (required),
`api_host`. **No outputs** — it's a pure side-effecting delete.

### `neondatabase/reset-branch-action` (https://github.com/neondatabase/reset-branch-action)
Also `runs.using: composite`, also shells out to `neonctl@2.22.0`. Its actual run script (read
verbatim from `action.yml`):
```bash
reset_args=("branches" "reset" "$INPUT_BRANCH" "--project-id" "$INPUT_PROJECT_ID")
if [[ -n "$INPUT_PARENT" ]] && [[ "$INPUT_PARENT" != "false" ]]; then
  reset_args+=("--parent")
fi
reset_args+=("--output" "json")
neonctl "${reset_args[@]}" > branch_out
branch_id=$(jq --raw-output '.id' < branch_out)
# ...then, separately:
neonctl cs "$branch_id" --project-id "$INPUT_PROJECT_ID" [--role-name ...] [--database-name ...] \
  [--ssl ...] [--prisma ...] --extended -o json [--pooled]
# db_url = the above JSON's .connection_string; repeated with --pooled for db_url_with_pooler
```
This is the clearest primary-source confirmation of the connection-string mechanics (see §5):
**`neonctl branches reset` itself does not return a connection string** — the action calls
`neonctl branches reset` to get a `branch_id`, then makes a *second*, separate call to `neonctl
cs <branch_id> --extended -o json` to fetch the connection string, host, and password as JSON
and parses `.connection_string`/`.host`/`.password` out of it with `jq`.
Inputs: `project_id` (required), `branch` (required — name or ID to reset), `api_key`
(required), `api_host`, `parent` (required to actually get a parent-tip reset — see §2's "only
supported reset operation" caveat), `cs_role_name`, `cs_database`, `cs_prisma`, `cs_ssl`.
Outputs: `branch_id`, `db_url`, `db_url_with_pooler`, `host`, `host_with_pooler`, `password`.

### `neondatabase/schema-diff-action` (https://github.com/neondatabase/schema-diff-action)
Per its own README/marketplace listing: computes a schema diff between `compare_branch` and
(by default) its parent, or an explicit `base_branch`, and posts/updates a PR comment with the
diff. Requires `project_id`, a Neon API key secret, and job permissions `pull-requests: write`,
`contents: read`. Not directly relevant to running migrations, but relevant to a "PR touches
`sql/schema/`" review workflow if this repo wants schema-diff-on-PR as a separate concern from
branch provisioning.

---

## 4. Neon's documented "ephemeral branch per PR" pattern

Neon's own `docs/guides/branching-github-actions` overview page
(https://neon.com/docs/guides/branching-github-actions) is largely a directory of the four
actions above plus links to example starter repos (Vercel, Cloudflare Pages, Fly.io
integrations) — it does not itself inline a full example workflow. The actual worked example
was read directly from Neon's own example repo,
**`neondatabase/preview-branches-with-vercel`** (`gh api
repos/neondatabase/preview-branches-with-vercel/contents/.github/workflows/*`), which is the
concrete, primary-source shape of the pattern:

`.github/workflows/deploy-preview.yml` (trigger: `on: [pull_request]` — i.e. the default
`opened`/`synchronize`/`reopened` events):
```yaml
- name: Create Neon Branch
  id: create-branch
  uses: neondatabase/create-branch-action@v5
  with:
    project_id: ${{ env.NEON_PROJECT_ID }}
    # parent: dev # optional (defaults to your primary branch)
    branch_name: preview/pr-${{ github.event.number }}-${{ steps.branch-name.outputs.current_branch }}
    username: ${{ env.NEON_DATABASE_USERNAME }}
    database: ${{ env.NEON_DATABASE_NAME }}
    api_key: ${{ env.NEON_API_KEY }}

- name: Run Migrations
  run: |
    echo DATABASE_URL=${{ steps.create-branch.outputs.db_url_with_pooler }} >> .env
    echo DIRECT_URL=${{ steps.create-branch.outputs.db_url }} >> .env
    npx prisma generate
    npx prisma migrate deploy
```
Note the pattern of using the **pooled** output (`db_url_pooled`/`db_url_with_pooler`) for the
app's general `DATABASE_URL` and the **direct/unpooled** output (`db_url`) as `DIRECT_URL`
specifically for the migration tool — Prisma's own convention, but structurally identical to
this repo's existing "migrations and long-running pool both need a DSN, but not necessarily the
*same kind* of DSN" situation (see §5, and CODING_STANDARDS.md's direct-vs-pooler note).

`.github/workflows/cleanup-preview.yml` (trigger: `on: pull_request: types: [closed]`):
```yaml
- name: Delete Neon Branch
  uses: neondatabase/delete-branch-action@v3.1.3
  with:
    project_id: ${{ secrets.NEON_PROJECT_ID }}
    branch: preview/pr-${{ github.event.number }}-${{ github.event.pull_request.head.ref }}
    api_key: ${{ secrets.NEON_API_KEY }}
```

**Naming convention**: Neon's docs (via search-indexed guide text cross-checked against the
example repo above, which matches it exactly) recommend `preview/pr-<pull_request_number>-<git-branch-name>`
so the ephemeral branch is unambiguously tied back to the PR that spawned it — this is exactly
the naming both `create-branch-action` (branch created) and `delete-branch-action` (branch
looked up by the *same* computed name to delete) rely on; there's no separate ID persisted
between the two jobs in this example — the name itself is the join key. (This matters for
Open Questions below re: keying on PR number vs. commit SHA.)

Overall shape confirmed by this primary source: **branch created on PR open/sync, migrations
run against it as its own CI step immediately after creation, branch deleted on PR close** —
matches exactly what the task description expected Neon to document, and it's Neon's own
example repo (not a third party) doing it.

---

## 5. Connection strings / credentials on a new branch

**Roles/passwords are inherited from the parent, automatically**, per
https://neon.com/docs/manage/branches: "When creating a new branch, the branch will have the
same Postgres roles and passwords as the parent branch." So a fresh branch does not require
creating new roles or new passwords by default.

**Exception: protected-branch children get regenerated passwords.** Same page: "New passwords
are automatically generated for Postgres roles on branches created from protected branches" —
a deliberate security measure specific to Neon's "protected branch" feature (relevant if this
repo's production branch is ever marked protected).

**But the connection string itself still has to be fetched fresh**, because a branch's
connection string is tied to its own compute endpoint, not just its role/password: "Connecting
to a database in a branch requires connecting via a compute associated with the branch." Same
host/role/password don't automatically resolve to a working DSN for the new branch — the
*endpoint hostname* differs per branch even if the role name and password are unchanged.

**Mechanism to obtain it programmatically in a CI step** — two distinct, both primary-source-
confirmed patterns, not one canonical answer:
1. **`create-branch-action`'s own outputs** — `db_url`/`db_url_pooled` are emitted directly as
   action step outputs (§3) — the action itself calls the Neon API and returns a ready DSN, no
   separate lookup step needed for a *newly created* branch.
2. **`neonctl cs <branch-or-id> --extended -o json`** (the `connection-string` command, short
   alias `cs`) — this is what `reset-branch-action` shells out to internally (§3) to get a DSN
   for a branch that already exists (post-reset), parsing `.connection_string`/`.host`/
   `.password` from its JSON output. This is the general-purpose mechanism for "I have a branch
   ID/name, I need its DSN" outside of the create-action's own bundled output — i.e., this is
   what a hand-rolled workflow step (not using `create-branch-action`) would call to get a DSN
   to pipe into `-db-dsn` for this repo's `cmd/migrate`/`cmd/blog`.

Both mechanisms return a **direct (unpooled)** connection string by default, and a **pooled**
variant on request (`db_url_pooled` / `neonctl cs --pooled`) — matching this repo's existing
direct-vs-pooler distinction (CODING_STANDARDS.md: "Use Neon's direct (unpooled) connection
string, not its PgBouncer pooler endpoint" for both binaries' own `pgxpool` pools). A CI step
provisioning a branch for `cmd/migrate`/`cmd/blog` to run against should use the unpooled
`db_url` output for the same reason production does — both are long-running/one-shot processes
managing their own connection lifecycle, not serverless functions needing PgBouncer's
connection multiplexing.

---

## 6. Cost / lifecycle: autosuspend and branch-count limits

**Autosuspend (scale-to-zero) is a per-compute-endpoint setting, not a project-global one**,
confirmed via https://neon.com/docs/guides/scale-to-zero-guide: it's configured "On the
**Computes** tab" of a specific branch ("Select a branch. On the Computes tab, click Edit."),
and the Neon API's "Update compute endpoint" call takes a specific `endpoint_id` — each branch's
compute endpoint carries its own `suspend_timeout_seconds`. A project-level "Update project"
API call only "sets a default for all compute endpoints created **in the future**; it does not
change the configuration of existing computes" — so a newly created ephemeral branch's compute
inherits whatever the *project's current default* suspend timeout is unless overridden per-
branch (or via `create-branch-action`'s own `suspend_timeout` input, §3, which lets a CI-created
branch set this explicitly at creation time rather than relying on the project default).

**Default timeout**: per https://neon.com/docs/introduction/scale-to-zero, "Neon compute scales
to zero after an inactive period of 5 minutes." On the Free plan this is fixed; paid plans can
disable scale-to-zero (always-on compute) or (per the scale-to-zero guide) adjust the timeout.
Scale-to-zero only applies to computes up to 16 CU — larger computes stay always-on. Once a
suspended compute is queried again, it "reactivates automatically within a few hundred
milliseconds" (cold start).

**Branch-count limits**, per https://neon.com/docs/introduction/plans (fetched directly): **Free
plan — 10 branches/project; Launch plan — 10 branches/project; Scale plan — 25 branches/
project.** Extra branches beyond a paid plan's included allowance are billed at **$1.50/branch-
month, prorated hourly** (≈$0.002/hour) rather than hard-capped — but the Free plan has no
"extra branches" option at all: branch creation simply fails past 10 until one is deleted or the
project is upgraded. Both paid tiers support up to "5,000 branches/project" as a theoretical
ceiling, with a documented path to request a higher limit via the Console feedback form if
needed. (Compute cost separately: Free plan includes 100 CU-hours/project/month; Launch bills
$0.106/CU-hour, Scale $0.222/CU-hour, on top of/instead of the included allowance.)

**Implication for a "one branch per open PR" or "one branch per dev-build" pattern**: on the
Free plan, 10 concurrent branches is a real ceiling that a naive "never clean up" ephemeral-
branch workflow could hit fairly quickly (production branch + a handful of open PRs/dev builds
already eats into that budget) — the delete-on-close step in §4's pattern isn't just tidiness,
it's what keeps the workflow from hitting this limit. A "per commit to dev" scheme (as opposed
to per-PR, which naturally caps at "however many PRs are open at once") would need its own
explicit cleanup step (e.g., reset the *same* branch on each new commit via `reset-branch-
action`, rather than creating a new branch per commit) to avoid the same ceiling — see Open
Questions.

---

## Open questions for this repo's design

Flagged for the follow-up design/grilling conversation — not resolved here:

1. **Key ephemeral branches on the dev-branch's HEAD commit vs. PR number?** Neon's own worked
   example (§4) keys the branch name on PR number (`preview/pr-<number>-<head-ref>`), which
   naturally caps concurrent ephemeral branches at "however many PRs are currently open." A
   per-commit scheme (rebuild on every push to a long-lived `dev` branch, per
   `docs/research/release-pipeline.md`'s still-unresolved dev-channel discussion) has no such
   natural cap unless it either (a) reuses/resets one fixed branch per logical channel via
   `reset-branch-action` rather than minting a new branch per commit, or (b) adds its own
   explicit delete-superseded-branch step — otherwise it risks the 10-branch Free-plan ceiling
   from §6 in a way the PR-scoped pattern doesn't.

2. **Should `cmd/migrate` run automatically against a freshly created branch as part of a
   dev-flow CI job?** Neon's own example (§4) runs migrations (there, Prisma's) immediately
   after `create-branch-action`, using the unpooled `db_url` output. The direct structural
   analogue here is a CI step invoking this repo's actual `cmd/migrate` binary (already built as
   its own container image / init-container per CODING_STANDARDS.md's Migrations section)
   against the new branch's `db_url`, before `cmd/blog` (or its tests) run against the same
   branch. Whether this belongs in a PR-triggered workflow, a dev-branch-push workflow, or both
   is undecided.

3. **Does this replace or supplement `internal/models`'s existing `newTestDB(t)` pattern?**
   Worth flagging precisely: `CODING_STANDARDS.md`'s Testing section documents an aspirational
   `newTestDB(t)` helper ("running goose migrations from `sql/schema` against a scratch
   database... Neon branch or local instance"), but the actual current
   `internal/models/db_test.go` does **not** implement any such helper — `TestOpenPool_RealDatabase`
   just reads `BLOG_DB_DSN` directly from the environment (skipping if unset or in `-short`
   mode) and points it at whatever database that env var resolves to, which in this repo's
   checked-in `.envrc` is the **production** Neon connection string. A CI-provisioned ephemeral
   branch (via `create-branch-action` or a raw `neonctl branches create` + `neonctl cs` step)
   is the natural way to give `newTestDB(t)` a real, isolated, disposable Postgres instead of
   either (a) never actually building `newTestDB` and continuing to point integration tests at
   prod, or (b) requiring a local Postgres in every dev/CI environment. Whether that means CI
   provisions one branch per test run and exports it as `BLOG_DB_DSN` before `go test` — i.e.
   this becomes the mechanism that finally implements `newTestDB(t)` — or whether the two stay
   separate (ephemeral branches for a full running app, `newTestDB(t)` staying a distinct
   concern) is open.

4. **Interaction with the release-please/GoReleaser dev-channel Docker-image design.** These are
   largely orthogonal axes — Neon branching provisions a *database*, the dev-channel design
   (`docs/research/release-pipeline.md` §4/Open Question 7) provisions a *binary/image* — but
   they intersect wherever a dev-channel image is actually run/smoke-tested in CI: if the
   dev-channel workflow ever spins up the built `:dev` image and points it at a database, that
   database is a natural candidate to be a fresh Neon branch rather than shared prod, which
   ties this doc's §6 branch-count/cleanup concerns to whatever cadence that other doc's Open
   Question 1 (rebuild-on-every-commit vs. something coarser) lands on.
