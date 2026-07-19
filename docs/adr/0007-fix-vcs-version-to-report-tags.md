# Fix internal/vcs.Version() to report git tags, not just raw commit SHAs

`internal/vcs.Version()` currently prefers `vcs.revision` (Go's build-info setting, always a raw commit SHA) and only falls back to `bi.Main.Version` when that's empty — which it never is. This means the admin UI and `/health` never show a semver tag like `v0.2.0`, only a bare SHA (`+dirty` appended when the working tree doesn't match `HEAD`). It also inverts `docs/references/lets-go-further.html`'s own reference implementation of this exact package, which this repo's `CODING_STANDARDS.md` otherwise follows, without an ADR or comment explaining the deviation.

We fix this to match the book: prefer `bi.Main.Version`, which Go computes from git tags — exactly `v0.2.0` when `HEAD` is on that tag, and a pseudo-version embedding the commit hash when it's ahead of the last tag (e.g. pre-release `development`/feature-branch builds, where no tag exists yet). This gives tagged-release-shows-clean-tag / untagged-build-shows-commit-hash behavior for free, from the same mechanism, with no new CI tooling.

We also fix `.dockerignore` excluding `docs/`, `bin/`, `.devcontainer/` (all git-tracked) from the Docker build context: since `.git` itself is copied in full, those tracked-but-context-excluded files make `git status` inside the build container see a dirty tree on every build, tagged or not — a false `+dirty` unrelated to actual uncommitted changes.

## Considered Options

- Bypass Go's built-in VCS stamping entirely and inject the version via `-ldflags -X` computed in CI (the GoReleaser-style approach researched in `docs/research/release-pipeline.md`) — rejected: this pipeline doesn't run GoReleaser (both `release.yml` and `dev-image.yml` build via a plain `docker build`), so this would introduce new build machinery to fix a bug that Go's existing built-in mechanism, used correctly, already solves.
