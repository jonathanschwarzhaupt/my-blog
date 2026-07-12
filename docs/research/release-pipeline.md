# Research: release-please + GoReleaser release pipeline

Scope: primary-source findings on `release-please`, `GoReleaser`, and how they compose in
GitHub Actions, for a design discussion about this repo (`cmd/blog` + `cmd/migrate`, one
`internal/` tree, one Go module — see `CODING_STANDARDS.md` and `docs/adr/0001`–`0003`).
This repo has no CI/release workflows yet (`.github/workflows/` does not exist at research
time), so every recommendation below is greenfield, not a migration.

Research method: primary sources only — release-please's and GoReleaser's own docs/source
(fetched from `googleapis/release-please`, `googleapis/release-please-action`, and
`goreleaser/goreleaser` on GitHub, `main` branch, July 2026), not blog-post summaries.

---

## 1. release-please: mechanics, manifest config, and multi-package versioning

**Basic flow.** release-please parses Conventional Commits, maintains a standing "Release
PR" that bumps version(s)/CHANGELOG as commits land, and — when that PR is merged — tags a
release and (unless disabled) creates a GitHub Release.
Per `release-please-action`'s README: "These Release PRs are kept up-to-date as additional
work is merged. When you're ready to tag a release, simply merge the release PR."
(https://github.com/googleapis/release-please-action, `README.md`, "What's a Release PR?"
section, fetched from `raw.githubusercontent.com/googleapis/release-please-action/main/README.md`).

**Manifest mode = two source-controlled files.** Per release-please's own
`docs/manifest-releaser.md`: `release-please-config.json` ("releaser specific configuration
for all packages") and `.release-please-manifest.json` ("package version tracking"). "The
motivation of the manifest-based releaser is support for monorepos: a combined Release PR
will be created for all configured packages... release configuration for potentially
hundreds of libraries is combined in two configuration files."
(https://github.com/googleapis/release-please/blob/main/docs/manifest-releaser.md)

**`packages` map is the core config unit.** Each key is "the relative path from the repo
root to the folder that contains all the files for that package;" `.` is "a special case for
handling to root of the repository." Each entry can override `release-type`, `package-name`,
`changelog-path`, `changelog-host`, `exclude-paths`, etc. (same doc, "Configfile" section).

**Default behavior is independent-per-package versioning.** The manifest "will record a new
version into the manifest file for each package it is configured to release" — i.e., each
`packages` entry gets its own version number tracked independently in
`.release-please-manifest.json`, bumped according to that path's own Conventional Commits.
(same doc, "Manifest" section)

**Every top-level/per-package config field is enumerated in the JSON Schema**
(https://github.com/googleapis/release-please/blob/main/schemas/config.json, fetched
directly), including (verified by reading the raw schema, not just prose docs):
- `release-type` (per package, string — see release types below)
- `bump-minor-pre-major`, `bump-patch-for-minor-pre-major`, `prerelease-type`, `versioning`
- `release-as` ("[DEPRECATED] Override the next version of this package. Consider using a
  `Release-As` commit instead")
- `skip-github-release`, `skip-changelog`, `draft`, `force-tag-creation`, `prerelease`
- `include-component-in-tag` (default `true`), `include-v-in-tag` (default `true`),
  `include-v-in-release-name` (default `true`)
- `version-file`: "Path to the specialize [sic] version file. Used by `ruby` and `simple`
  strategies" (and, per source below, also honored by `go`)
- `extra-files`, `snapshot-label`, `initial-version`, `exclude-paths`

**Release (language) types**, per `docs/customizing.md`
(https://raw.githubusercontent.com/googleapis/release-please/main/docs/customizing.md):

| release-type | what it manages |
|---|---|
| `go` | "A repository with a CHANGELOG.md" (no version file by default) |
| `simple` | "A repository with a version.txt and a CHANGELOG.md" |
| `node`, `python`, `rust`, `java`, `helm`, `terraform-module`, etc. | language-specific manifest files (`package.json`, `Cargo.toml`, `pom.xml`, ...) |

Confirmed by reading the actual strategy source
(https://github.com/googleapis/release-please/blob/main/src/strategies/go.ts and
`.../simple.ts`, fetched raw): `Go.buildUpdates()` always pushes a `Changelog` update, and
*only* pushes a version-file update `if (this.versionFile)` is set (defaults to `''`, i.e.
off) — using a `VersionGo` updater. `Simple.buildUpdates()` always pushes both a `Changelog`
update *and* a version-file update, defaulting `versionFile` to `'version.txt'`, with
`createIfMissing: false` (the file must already exist in the repo for release-please to
touch it). Neither strategy invents a version-embedding mechanism specific to Go binaries —
embedding the version into the built binary is left entirely to the build tool (GoReleaser,
see §2), not to release-please.

**Root-path (`.`) releases are an explicit, named use case.** Per `docs/customizing.md`,
"Releasing Root Path of Library (`.`)": "`.` indicates a release should be created when any
changes are made to the codebase" — originally built so a googleapis monorepo could publish
individual libraries *and* a combined root release, but nothing about `.` requires other
packages to exist; a manifest config can legitimately have exactly one entry, `"." : {}`.

**release-please's own design doc recommends manifest mode even for a single package.** Per
`docs/design.md`'s "Monorepo support" section (fetched raw from
`raw.githubusercontent.com/googleapis/release-please/main/docs/design.md`): "We highly
recommend using manifest configurations (even for single library repositories) as the
configuration format is well defined (see schema) and it reduces the number of necessary API
calls. In fact, the original config options for `release-please` are actually converted into
a manifest configured release that only contains a single component." This means even the
simplest possible setup — the action's `release-type: simple` input with no config file at
all (see §5) — is internally a one-entry manifest release. There is no separate "non-manifest
codepath" to reason about.

**Mechanisms that make *multiple* packages share one version (not needed if you use a single
root package, but relevant to understand why they exist):**

- **`linked-versions` plugin** — the purpose-built mechanism for "I have N independently
  configured components/paths, but they must always carry the same version number." Per
  `docs/customizing.md`, "Plugins" → "linked-versions": "allows you to 'link' the versions of
  multiple components in your monorepo. When any component in the specified group is
  updated, we pick the highest version amongst the components and update all group
  components to the same version (keeping them in sync)." Config shape (from the same doc):
  ```json
  {
    "plugins": [
      { "type": "linked-versions", "groupName": "my group", "components": ["pkgA", "pkgB"] }
    ]
  }
  ```
  Read the plugin source directly
  (https://github.com/googleapis/release-please/blob/main/src/plugins/linked-versions.ts,
  fetched raw): it operates on `component` names (a per-package `component` config field),
  computes the `primaryVersion` as the max across the group's proposed versions, forces every
  group member's strategy to `releaseAs: primaryVersion.toString()`, and (by default,
  `merge: true`) merges their candidate PRs into one PR via the internal `Merge` plugin.
- **Workspace plugins** (`node-workspace`, `cargo-workspace`, `maven-workspace`) — solve a
  *different* problem: propagating internal dependency-version bumps between packages that
  depend on each other (e.g., patch-bump package B because package A, which B depends on,
  was bumped). They create version *coupling* via dependency graphs, not identical version
  numbers, and don't apply to this repo (`cmd/blog` and `cmd/migrate` don't depend on each
  other as packages).

**Conclusion for independent-vs-shared versioning:** `linked-versions` (and the workspace
plugins) exist to keep *separately configured* `packages` entries in sync. They solve a
problem this repo doesn't have, if the repo is configured with exactly one package (root
`.`), because there is then only one version to begin with — see §3.

---

## 2. GoReleaser: multi-binary builds, Docker images, CI integration

**One `builds:` entry per binary.** Per the Go builder doc
(https://goreleaser.com/customization/builds/ → `builders/go.md`, fetched raw from
`www/content/customization/builds/builders/go.md` in `goreleaser/goreleaser`), the documented
multi-binary example is exactly this repo's shape (N binaries from one module):
```yaml
builds:
  - main: ./cmd/cli
    id: "cli"
    binary: cli
    goos: [linux, darwin, windows]
  - main: ./cmd/worker
    id: "worker"
    binary: worker
    goos: [linux, darwin, windows]
  - main: ./cmd/tracker
    id: "tracker"
    binary: tracker
    goos: [linux, darwin, windows]
```
Each entry's `main` is "Path to main.go file or main package... Default: `.`" and `binary` is
"Binary name... Default: Project directory name." `id` defaults to "Project directory name"
too, and is used to cross-reference a build from `dockers`/`archives`/etc via `ids:`.

**Version injection is template-driven and identical across every build in one run.**
`ldflags` default value (from the same doc): `'-s -w -X main.version={{.Version}}
-X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser'`. `.Version`,
`.Commit`, etc. are GoReleaser *run-level* template variables (documented in
`www/content/customization/general/templates.md`, fetched raw — `.Version`: "the version
being released", `.Tag`: "the current git tag") — they're computed once per GoReleaser
invocation, not per `builds:` entry, so every binary built in the same `goreleaser release`
run automatically gets the same version string with zero extra config. There is no
per-build override needed to "keep them in sync" — they're in sync by construction, because
one `goreleaser release` run only ever sees one resolved version (the tag it was invoked
against).

**Version comes from the git tag.** Per `getting-started/how-it-works.md` (fetched raw):
GoReleaser expects "a clean working tree" and "a SemVer-compatible version (e.g.
`10.21.34-prerelease+buildmeta`)" — i.e. it reads the current tag via git, it does not read
version numbers out of any repo file (confirmed: no GoReleaser config key sets "the
version" directly for a real release — only `snapshot.version_template` overrides it for
non-tag runs, see §4).

**Docker image config: `dockers`/`docker_manifests` are being deprecated in favor of
`dockers_v2`.** Per `www/content/customization/package/docker.md` (fetched raw): "Phasing
out in v2.12. Docker Images v2 is preferred instead." The classic two-stanza shape (still
documented, still functional) is:
```yaml
dockers:
  - image_templates: ["myuser/myimage:{{ .Tag }}", "myuser/myimage:v{{ .Major }}", "myuser/myimage:latest"]
docker_manifests:
  - name_template: "foo/bar:{{ .Version }}"
    image_templates: ["foo/bar:{{ .Version }}-amd64", "foo/bar:{{ .Version }}-arm64v8"]
```
`dockers` builds one image per architecture (matched against `builds` output by `goos`/
`goarch`/`ids`); `docker_manifests` fuses same-tag, different-arch images into one
multi-arch manifest reference via `docker manifest create`+`push` (same doc, "How it
works": "we basically build and push our images as usual, but we also add a new section...
defining[/] which images are part of which manifests. GoReleaser will create and publish
the manifest in its publishing phase.") — note: **the per-arch images must already be
pushed** for `docker manifest create` to succeed (documented limitation, linked to
https://github.com/goreleaser/goreleaser/issues/2606 in the same doc).

**`dockers_v2` (current recommended path, in GoReleaser ≥ v2.12, still labeled
experimental)** uses `docker buildx` directly and is a single stanza (no separate manifest
step): `images:` (list of image names), `tags:` (list, template-able), `platforms:` (default
`[linux/amd64, linux/arm64]`). It builds one true multi-arch manifest per run. Per
`www/content/customization/package/dockers_v2.md` (fetched raw): "The `dockers_v2` name is
provisional. It will replace `dockers` and `docker_manifests` in GoReleaser v3 (no ETA), and
will then be simply `dockers`." Since this repo has no existing `dockers`/`docker_manifests`
config to preserve compatibility with, `dockers_v2` is the more future-proof starting point
for a new pipeline, at the cost of being labeled experimental.

**GitHub Actions integration (`goreleaser-action`).** Per
`www/content/customization/ci/actions.md` (fetched raw), the canonical workflow triggers on
tag push:
```yaml
on:
  pull_request:
  push:
    tags: ["*"]
permissions: { contents: write }
jobs:
  goreleaser:
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }   # required: GoReleaser needs full git history
      - uses: actions/setup-go@v5
        with: { go-version: stable }
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }
```
Explicit warnings in that doc: "GoReleaser Action will not install nor setup any other
software needed to release" (Docker login, GPG, etc. are the user's job), and `fetch-depth:
0` is required because GoReleaser needs full tag/commit history to compute changelogs and
previous-version deltas.

---

## 3. Keeping two binaries (`cmd/blog`, `cmd/migrate`) on ONE version

Given §1 and §2, the shape that fits "one version, N binaries" is:

- **release-please side:** a manifest config (`release-please-config.json`) with a **single
  package entry at the repository root**, `"." : { "release-type": "go" }` (or `"simple"`),
  and a matching `.release-please-manifest.json` with one key, `"."`. Per §1's design-doc
  citation, this is not a special/reduced mode — it's the same manifest engine as a
  100-package monorepo, just with one entry, and it is explicitly the recommended shape even
  for single-artifact repos.
- Because there is only **one** `packages` entry, there is only one version being tracked —
  `linked-versions` (or any workspace plugin) is solving a problem (N independently
  versioned components that must be forced to move together) that doesn't arise here. Per
  the `linked-versions` source read in §1, that plugin operates on `component` names across
  *multiple* strategy instances; with one package there is nothing to link.
- **GoReleaser side:** one `.goreleaser.yaml`, `builds:` with two entries (`main: ./cmd/blog`,
  `main: ./cmd/migrate`), each with its own `id`/`binary`. Both are built from the same
  invocation of `goreleaser release`, against the same resolved git tag, so both automatically
  embed the same `{{.Version}}` via `ldflags` (§2) — no linking mechanism needed on the
  GoReleaser side either.
- **The tag itself**: release-please's default tag format for a root/`.`-path package is
  `v<version>` (no component prefix), controlled by `include-component-in-tag` (schema
  default `true`, but per `docs/manifest-releaser.md`'s "Subsequent Versions" section, a
  root package with no distinguishing component name naturally produces a plain `v1.2.3`-
  style tag once `include-component-in-tag` is set `false` if the default component-prefixed
  form doesn't match — this needs to be checked against actual generated tag output, see
  Open Questions).

This is a deliberate one-package-at-root design choice, not GoReleaser or release-please
"noticing" the two binaries share a version — the sharing comes entirely from there being
one release-please component and one GoReleaser invocation per tag.

---

## 4. Docker tagging: tagged releases vs. a floating dev/edge channel

**Tagged release channel** (`dockers`/`docker_manifests`, or `dockers_v2`, per §2): image
tags derived from the resolved semver tag, e.g. `image_templates: ["myuser/myimage:{{
.Tag }}", "myuser/myimage:v{{ .Major }}", "myuser/myimage:latest"]` — "Keeping docker images
updated for current major" example in `package/docker.md`, generates `v1.6.4`, `v1`, `v1.6`,
`latest` from a single build when tag `v1.6.4` is released.

**GoReleaser's `--snapshot` flag is explicitly NOT a publish mechanism.** Read directly from
GoReleaser's own CLI source
(https://github.com/goreleaser/goreleaser/blob/main/cmd/release.go, fetched via shallow
clone): the `--snapshot` flag is registered with the description "Generate an unversioned
snapshot release, skipping all validations and without publishing any artifacts (**implies
--skip=announce,publish,validate**)." This is confirmed by the docs too
(`www/content/customization/publish/snapshots.md`, fetched raw): "Note that the idea behind
GoReleaser's snapshots is for local builds or to validate your build on the CI pipeline.
Artifacts won't be uploaded and will only be generated into the `dist` directory." So
`goreleaser release --snapshot` cannot be the mechanism that pushes a `:dev`/`:edge` image to
a registry on every branch push — publishing is unconditionally skipped in snapshot mode.
`snapshot.version_template` (default `{{ .Version }}-SNAPSHOT-{{.ShortCommit}}`) only affects
the *locally generated, unpublished* artifact's version string.

**`goreleaser build` is narrower still.** Per its own `--help` text in source
(`cmd/build.go`, fetched via shallow clone): "The `goreleaser build` command is analogous to
the `go build` command, in the sense it only builds binaries" — no archives, no packages, no
Docker, no publish step exists in this command at all, snapshot or not.

**GoReleaser's actual rolling/pre-release mechanism, `--nightly`, is GoReleaser Pro-only.**
Per `www/content/customization/publish/nightlies.md` (fetched raw, note the `{{<
g_featpro >}}` marker in the source before the prose even starts): "Whether you need beta
builds or a rolling-release system, the nightly builds feature will do it for you. To enable
it, you must use the `--nightly` flag." It supports `nightly.tag_name` (a fixed floating tag,
e.g. `devel`), `keep_single_release: true` (deletes the previous pre-release under that same
tag before publishing the new one), and does publish (unlike `--snapshot`) if
`publish_release: true`. This would be the natural GoReleaser-native answer to "one floating
dev/edge channel, rebuilt on every push" — **but it is a paid Pro feature**, not available in
OSS GoReleaser.

**Implication for a dev/edge Docker channel with OSS GoReleaser:** since (a) the tag-driven
release flow assumes a real tag exists, (b) `--snapshot` deliberately never publishes
anything, and (c) the one feature designed for exactly this (`--nightly`) is Pro-only, a
free/OSS pipeline for a floating pre-release image channel most likely has to either (i) not
route through GoReleaser's Docker publish stanzas at all for that channel — a separate,
plain `docker/build-push-action`-style step in a branch-push-triggered workflow, tagging
`:dev`/`:edge` directly — optionally still using `goreleaser build --snapshot` just to
produce the binaries to embed in that image, or (ii) create a real (if synthetic/moving)
pre-release git tag on every push to the dev branch and run the normal tag-triggered
GoReleaser release flow against it (this is broadly how nightly/rolling releases work in
projects that don't have GoReleaser Pro, but note it litters the tag list unless cleaned up,
and interacts with release-please's own tag/version bookkeeping — see Open Questions).

---

## 5. Composing release-please + GoReleaser in one pipeline

**The canonical shape is two workflows, split by trigger.**

**Workflow A — release-please, triggered on every push to `main`.** Documented pattern (from
`release-please-action`'s README, "Basic Configuration", fetched raw):
```yaml
on:
  push:
    branches: [main]
permissions: { contents: write, issues: write, pull-requests: write }
jobs:
  release-please:
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          token: ${{ secrets.MY_RELEASE_PLEASE_TOKEN }}
          release-type: simple
```
For "advanced" (manifest) config, the README says simply: "set up a manifest config and then
configure this action" with `config-file`/`manifest-file` inputs pointing at
`release-please-config.json`/`.release-please-manifest.json` (defaults already match those
filenames, so often no override is needed at all).

Important operational note from the same README, "Other Actions on Release Please PRs": the
built-in `GITHUB_TOKEN` will *not* trigger downstream workflows on the release-please PR or
on the tag/release it creates ("events triggered by the `GITHUB_TOKEN` will not create a new
workflow run" — GitHub's own recursive-workflow guard) — "if you want GitHub Actions CI
checks to run on Release Please PRs" (and, relevantly, if you want the *tag push* from
merging the release PR to trigger Workflow B below), you need a PAT/fine-grained token in
place of the default `GITHUB_TOKEN`. This is a real gotcha for the two-workflow composition:
**with the default token, tag-push-triggered Workflow B would never fire**, because
release-please's own tag push wouldn't count as a user-originated push event.

**Workflow B — GoReleaser, triggered on tag push.** Per §2's GoReleaser Actions doc: `on:
push: tags: ["*"]`, running `goreleaser/goreleaser-action@v7` with `args: release --clean`.

**Action outputs make this composable without a second workflow, too** — release-please-
action exposes `releases_created`/`release_created` (root-scoped) and `tag_name` as step
outputs (README, "Outputs" section), so a single workflow *can* conditionally run GoReleaser
in a later job/step gated on `if: steps.release.outputs.release_created`, checking out the
newly created tag. release-please-action's own README example does exactly this shape for
npm publication (a later step gated on `steps.release.outputs.releases_created`) rather than
a second workflow — so "one workflow with a conditional downstream job" and "two workflows,
one tag-triggered" are both patterns explicitly demonstrated by the two projects' own docs,
not just one canonical answer.

---

## Open questions this research surfaces for this specific repo

These are decisions for the follow-up design/grilling conversation, not resolved here:

1. **Root-mode vs. explicit one-entry manifest mode.** The action's bare `release-type:
   simple` input (no config file) and a manifest config with a single `"."` package are, per
   release-please's own design doc, the same underlying mechanism. Is there a reason to
   write the explicit `release-please-config.json` anyway (e.g., to also set `version-file`
   so the resolved version is checked into the repo somewhere, or to pin `include-v-in-tag`/
   `include-component-in-tag` behavior explicitly rather than relying on defaults)?

2. **`go` vs `simple` release-type for the root package.** `go` only touches
   `CHANGELOG.md` unless a `version-file` is explicitly set (no version file exists in this
   repo currently); `simple` expects a pre-existing `version.txt` (and will not create one —
   `createIfMissing: false`). Does this repo want a checked-in version file at all, given
   GoReleaser gets the version from the git tag regardless (§2), or is `CHANGELOG.md`-only
   sufficient?

3. **Tag shape**: does `include-component-in-tag: false` need to be set explicitly to get a
   plain `v1.2.3` tag (vs. some `component-v1.2.3` form) for a root-only package — this
   repo's actual generated tag format should be verified against a real release-please run
   rather than inferred from schema defaults alone.

4. **One workflow vs. two.** Both release-please-action's own docs (conditional job in the
   same workflow) and GoReleaser's own docs (separate tag-triggered workflow) are legitimate,
   documented patterns. Which fits this repo's preference for workflow simplicity vs.
   separation of concerns?

5. **Token for cross-workflow triggering.** If two workflows are chosen, the default
   `GITHUB_TOKEN` will not let release-please's merge-triggered tag push fire the
   GoReleaser workflow (GitHub's recursive-workflow guard) — a PAT or GitHub App token is
   required for release-please-action's `token` input. Where should that credential live
   given this is a personal/homelab project (ADR-0002)?

6. **Docker image stanza: `dockers`+`docker_manifests` (stable, but deprecated starting
   v2.12) vs. `dockers_v2` (current direction, still labeled experimental at time of
   research).** Since there's no existing config to preserve, starting on `dockers_v2` avoids
   a future migration, but "experimental" is a real caveat for a homelab deploy target.

7. **Dev/edge channel design is unresolved by GoReleaser OSS.** Given `--nightly` is Pro-only
   and `--snapshot` never publishes, the dev-branch image channel likely needs to bypass
   GoReleaser's own Docker publish stanzas entirely (plain `docker build`+`push` step in a
   branch-triggered workflow) rather than trying to force GoReleaser into that role. If so:
   what should the floating tag be named (`:dev`, `:edge`, `:main`?), does it need multi-arch
   support day one, and should it embed a `goreleaser build --snapshot`-produced binary or a
   plain `go build` one (simpler, no GoReleaser dependency for that path at all)?

8. **Interaction between a synthetic dev-channel tag (if Open Question 7 goes that route) and
   release-please's own tag/version bookkeeping** — release-please walks tag history to find
   "the last release," so any non-release tags pushed by a dev workflow would need to be
   excluded via `git.ignore_tags`/`ignore_tag_prefixes` (GoReleaser side, if such tags also
   feed a GoReleaser run) or otherwise kept out of release-please's view, to avoid corrupting
   changelog/version computation.
