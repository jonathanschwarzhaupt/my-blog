# Push a real git tag for dev-image builds, triggered by PR into main

`dev-image.yml` currently builds on every push to `development`, and the running binary's version (`internal/vcs.Version()`, fixed in ADR-0007) only reports a real tag when one exists at the built commit — dev builds have never had one, so they've always shown a pseudo-version/SHA. We change two things together:

**Trigger**: `push: branches: [development]` → `pull_request: types: [opened, synchronize], branches: [main]`, matching `migration-check.yml`'s existing pattern. This also serves a workflow goal independent of versioning: a `development`→`main` PR is opened once a batch of smaller per-issue merges into `development` amounts to one complete feature, so the dev image (and the homelab admin-preview deployment tracking it) refreshes once per feature, not once per issue — `opened` builds it, `synchronize` (any further push to `development` while that PR stays open, e.g. a fix-up merge) rebuilds it, with no separate trigger needed for the two cases.

**Versioning**: push an actual git tag, `v{version}-dev.{run_number}.{short_sha}` (e.g. `v0.3.0-dev.5.195e47c`), to the PR's real head commit before building. `internal/vcs.Version()` then reports that exact string, matching the Docker image tag (`{version}-dev.{run_number}.{short_sha}`, unprefixed, unchanged) with no new injection mechanism — the same `bi.Main.Version` route ADR-0007 already established, just with a tag now present for dev builds too.

## Why this doesn't corrupt release-please's bookkeeping

The obvious worry: could a `v0.3.0-dev.5.195e47c`-shaped tag ever get misread by release-please as a real release when it later scans tags on `main`? Traced release-please's actual `latestReleaseVersion()` (`src/manifest.ts`): a tag is only a release candidate if `commitShas.has(tag.sha)`, where `commitShas` is the set of commits reachable from the branch being scanned (`main`).

This repo's `development`→`main` merge is a **rebase**, not a real merge (`CODING_STANDARDS.md`'s branching table) — rebasing always mints new commit SHAs on `main`. A tag pushed to a `development`-branch commit therefore points at a SHA that can never become reachable from `main`'s history; the rebase orphans it from `main`'s perspective the moment it happens. So the dev tag is safe from collision by construction, not because of any naming trick — it can use the exact bare `vX.Y.Z...` shape Go's module versioning needs, with no distinguishing prefix required (a prefix would in fact break Go's own tag recognition for the root module).

## Other implementation notes

- **Checkout ref**: a `pull_request`-triggered workflow's default ref/`GITHUB_SHA` is GitHub's ephemeral test-merge commit, not the PR's real head. `actions/checkout` must be pinned to `ref: github.event.pull_request.head.sha` (and `short_sha` computed from that same SHA) — otherwise the tag and image would correspond to a synthetic commit not really part of `development`'s history.
- **Permissions**: pushing a tag needs `contents: write`, which `dev-image.yml` doesn't currently have (it only reads).
- **Floating `:dev` tag**: kept. Its meaning shifts from "latest `development` push" to "latest open release-candidate build," but it remains the one stable pointer the homelab deployment tracks without needing the exact per-build tag each time.
- **Tag lifecycle**: dev tags are never cleaned up. Volume is low (roughly one per feature-PR, plus occasional fix-up pushes) — not worth the added workflow logic of tracking which tags belong to which PR to prune them.
- **Dry-run target branch unchanged**: the existing `--target-branch=development` release-please dry-run (to compute the next version) stays correct — the conventional-commit content it's evaluating is the same whether read from `development`'s tip or from `main` post-merge, only the commit SHAs differ (via the rebase).
