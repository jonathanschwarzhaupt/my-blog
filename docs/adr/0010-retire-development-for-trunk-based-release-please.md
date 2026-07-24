# Retire the persistent development branch; main becomes the sole trunk release-please watches directly

The persistent `development` branch and `main` kept ending up with duplicate-content-but-different-SHA copies of the same commits (confirmed directly: `git diff origin/main origin/development` showed an empty tree diff while `git log` showed 9 "different" commits on each side — every one of them a content-identical twin). Investigating why led to two findings that both point the same direction:

**release-please's own design assumes exactly one branch.** Its docs (`docs/design.md`, fetched from `googleapis/release-please`) describe the whole lifecycle as: "A commit is merged/pushed to the release branch" → release-please maintains one rolling "release pull request" against that same branch → merging it cuts the release. There is no supported pattern for running it against two branches that periodically resync; `--target-branch` exists only for testing config changes in isolation (`docs/troubleshooting.md`), not for production use.

**Community consensus for automated, conventional-commits-driven release pipelines is trunk-based/GitHub Flow, not a persistent-development-branch (GitFlow-shaped) model.** GitFlow's long-lived parallel branches were designed for software shipped on a schedule (boxed software, app-store review); trunk-based is designed for continuous, automated releases — which is what this repo's release-please + dev-image pipeline actually is.

This repo's `development` branch was a GitFlow-shaped addition bolted onto a tool designed for the other model. That mismatch — not a mistake in any single sync — is what produced the divergence.

## Decision

`main` becomes the sole permanent branch. release-please watches it directly (unchanged config). `development` is retired.

**Short-lived `story/<id>-<slug>` branches replace `development`'s batching role**, without reintroducing a permanent second branch:

- A story/PRD that spans multiple issues gets one `story/<id>-<slug>` branch, created fresh off `main`. Sub-task `feature/*` branches squash-merge into it, same as they squash-merge into `development` today.
- When the story is complete, **one** PR from the story branch into `main`, **rebase-merged** (not squashed) — same reasoning already documented for `development`→`main`: squashing the whole story into one commit would collapse release-please's changelog into one undifferentiated line; rebasing replays each sub-task's own commit individually, preserving one changelog entry per issue.
- The branch is deleted immediately after merging. It is never reused and never persists alongside `main`.
- A standalone task that isn't part of a bigger story skips the story-branch layer entirely: a plain `feature/*` branch off `main`, PR'd straight into `main`.

**Why this doesn't recreate the divergence**: the old problem wasn't rebase-merging itself — it was that `development` kept *existing* afterward, permanently holding its own original-SHA copies of commits `main` had just received rebased copies of. A story branch is used once and deleted; there is no second permanent branch left to accumulate stale twins. `main` is the only survivor, every time.

**Existing CI needs almost no change.** The dev-image build (`.github/workflows/dev-image.yml`, `pull_request: types: [opened, synchronize], branches: [main]`, from ADR-0008) already just watches "any PR into `main`" — a story branch's PR (or a standalone task's PR) satisfies that trigger with zero modification. `release.yml` already watches pushes to `main`. The only real code change is the version dry-run step, which currently reads `--target-branch=development`; it needs to become `--target-branch=main`, since `development` no longer exists.

**Renamed "dev" → "rc" throughout** (workflow display name, tag prefixes `X.Y.Z-dev.N.sha` → `X.Y.Z-rc.N.sha`, floating `:dev` → `:rc`) — matches what these builds actually are in this model: release candidates you soak-test before promoting, not "whatever `development` currently contains." Promoting a candidate to an official release is exactly merging release-please's own rolling Release PR — since it always reflects everything merged to `main` since the last real release, including whatever commit your candidate build already validated, "promote" is a relabeling of something already tested, not a rebuild.

## Candidate git tags remain safe here — verified against release-please's actual source, not assumed

ADR-0008's safety argument for pushing a real git tag on every candidate build relied on `development`→`main` being a rebase, which made tagged `development` commits permanently unreachable from `main` afterward. In this new model, a candidate-tagged commit on `main` (or a merged story branch) stays *permanently* reachable — so that specific argument no longer applies, and needed re-checking, not just carrying forward.

Traced `latestReleaseVersion()` in release-please's `src/manifest.ts`. It resolves "what was the last release" from three sources, **in order, stopping at the first that finds anything**:

1. Merged PRs whose head-branch name matches release-please's *own* internally-generated branch-naming pattern (`BranchName.parse`) — a candidate tag pushed by a separate CI step was never created via that exact PR shape, so it's invisible here regardless of its name.
2. Actual GitHub Release objects (`github.releaseIterator()`) — a plain `git tag` with no corresponding GitHub Release doesn't appear here either.
3. Only if neither of the above finds *anything* — raw git tag scanning (`github.tagIterator()`), a bootstrapping fallback for a repo with zero release history.

This repo already has real releases (v0.2.0 onward), so step 2 always resolves and step 3 never runs. A candidate build's tag — as long as it's never wrapped in an actual GitHub Release (`gh release create`), just `git tag` + push — is structurally invisible to this bookkeeping, no matter how long it stays reachable or what it's named. Confirmed additionally: `Version.compare()` (`src/version.ts`) delegates to the real `semver` package, so even in the hypothetical case a candidate tag were somehow considered, a `-rc.N` suffixed version correctly sorts below its corresponding clean release per the semver spec.

Conclusion: keep pushing a real git tag per candidate build (unchanged mechanism from ADR-0008), just renamed to the `-rc.` scheme.

## Considered Options

- **Keep both branches, sync via fast-forward-only** (never rebase, always fast-forward `main` to `development`'s tip, then fast-forward `development` back to `main`'s tip after every release) — rejected: works, but is an ongoing discipline requiring a manual step after every single release, forever. Retiring `development` removes the failure mode instead of managing it.
- **Pure trunk-based with no batching mechanism** (every `feature/*` branch merges straight to `main`, one candidate build per merge) — rejected: loses "one candidate build per completed story," which the user explicitly wants; a multi-issue PRD would produce one build per sub-task instead of one for the finished story.
- **Switch candidate versioning to `-ldflags` injection instead of a real git tag** — rejected after verification: the tag-based approach (ADR-0007/ADR-0008's mechanism) is confirmed safe in this new model too (see above), so there's no reason to add `-ldflags` build machinery to solve a problem that turned out not to exist.

## Status of prior ADRs

Supersedes ADR-0008 specifically on the trigger context (development→main PR) and the tag-safety argument (rebase-erasure) — the candidate-build mechanism itself (real git tag, PR-triggered build) carries forward unchanged, just re-grounded in the reasoning above and renamed dev→rc.

## Implementation not yet done

This ADR records the decision. As of writing, `development` still exists and the workflows still reference it — the actual migration (retarget `dev-image.yml`'s dry-run to `--target-branch=main`, rename dev→rc throughout, delete the `development` branch once everything on it has landed on `main`, update branch-protection settings if any) is tracked separately and hasn't happened yet. `CODING_STANDARDS.md` has been updated to describe this as the current intended strategy despite that gap, so it doesn't drift back out of sync with the decision the way the old three-branch section did.
