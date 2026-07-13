# Neon database branches

This project uses more than one Neon branch of the same project. This note exists so
future-you (or whoever picks up the homelab repo's admin-preview deployment) doesn't have
to rediscover why, or hunt for where each branch's connection string lives.

## `production` (the default/primary branch)

The one real database. Both the stable public (Cloudflare Tunnel) and stable admin
(Tailscale-only) deployments described in ADR-0002/ADR-0003 point at this branch, updated
only when a tagged release is cut (see #44/#51). Its connection string lives in 1Password,
the same place `BLOG_DB_DSN` has always come from for local dev (`.envrc`).

## `development`

A persistent branch created off `production`, isolated from it — schema changes and
in-progress data from the always-on admin-preview environment (the one that auto-updates on
every push to the `development` git branch, per #44/#52) never touch live public data.
`cmd/migrate` runs for real against this branch as part of that deployment's own rollout,
same init-container pattern the production deployment already uses — not just a dry-run
check.

Its connection string is stored in 1Password (not committed anywhere in this repo, for the
obvious reason). Use the **direct** (unpooled) endpoint, not Neon's `-pooler` connection
string — this project's binaries need a direct connection (`cmd/migrate` in particular:
Neon's pooled/PgBouncer endpoint doesn't preserve session state across statements, which
breaks goose's session-level advisory lock).

## Ephemeral `preview/pr-<number>` branches

Created and destroyed automatically by `.github/workflows/migration-check.yml` (#48) — one
per open PR that touches migration-relevant paths, deleted on a successful check, left alive
on a failed one for debugging. Not persistent, not manually managed.

## Out of scope here

The actual Kubernetes Deployment that points the admin-preview environment at the
`development` branch's DSN lives in the separate homelab infrastructure repo — this repo's
job stops at the branch existing and its connection string being recorded somewhere
consumable.
