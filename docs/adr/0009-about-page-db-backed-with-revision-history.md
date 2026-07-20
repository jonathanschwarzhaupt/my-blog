# About page becomes DB-backed markdown with revision history; Skills become their own table

The About page is currently hardcoded prose in `ui/templ/pages/blog/about.templ` — editable only by changing code and redeploying. We make it editable from the admin UI, following two decisions made together because they were reasoned through in the same trade-off.

## The body becomes one markdown field, like a Post's

Everything except the fixed profile image and the Technical Skills block becomes a single markdown body (rendered through the same `internal/markdown`/goldmark path as a Post's body), editable from an admin form with no other fields — no title, no tags, no created-at; those are Post-specific concepts this page doesn't need. This means giving up per-section hand-authored HTML structure in favor of whatever headings the markdown itself contains.

## Skills become a table, not part of the markdown

The alternative — folding "Technical Skills" into the same markdown blob — was rejected: keeping the badge-chip visual styling would have meant inventing a bespoke markdown convention (e.g. a magic heading goldmark specifically recognizes) and custom rendering logic to detect it, with no precedent elsewhere in this codebase and real fragility (a slightly-off edit silently loses the styling). A `skill(id, category, name, order_key)` table costs more up front but reuses patterns this codebase already has end-to-end: a table + sqlc queries + an admin page, and `skillGroup` (the existing badge-chip templ component) keeps rendering exactly as it does today, just from queried rows instead of a hardcoded Go slice.

`category` is a free-text field on each Skill, not a separate curated entity — the About page groups Skills by whatever distinct category values exist, in the order each category is first encountered while iterating Skills by their own `order_key` (all Skills sharing a category render together as one block, wherever that category first appears — never split into multiple blocks even if not contiguous in `order_key`). No separate category-ordering concept exists; to move a whole category's block, adjust the `order_key` of its Skills. Categories aren't independently addable/renameable/reorderable as their own entity — keeping this to one flat ordered list of Skills, per the stated preference for the simplest solution that still lets category rendering stay dynamic.

Skills admin is a single "Manage Skills" page — one form listing every Skill (category, name, order), Save fully replaces the set (delete-all-then-reinsert), the same clear-and-resync pattern `syncPostProjects` already uses for a Post's Project associations. No per-Skill create/edit/delete plumbing.

**Fixed position**: Technical Skills renders after the markdown body, as the page's final section (profile image + intro stay fixed template chrome before the markdown; nothing renders after Skills). Skills can no longer be interleaved mid-prose the way it visually reads today, in exchange for not needing any in-markdown placement convention.

## Revision history, not just an edit-conflict guard

"Light versioning" resolved to: real, browsable, restorable history — not the version-counter-only pattern Posts/Projects use (which guards against a lost concurrent write but holds no history at all; see `CONTEXT.md`'s Revision/Version distinction). Implemented as an insert-only `about_revision(id, body, created_at)` table: every save inserts a new row, never overwrites one. The public About page always renders the latest row. Admin gets a history view (past revisions by timestamp) with Restore, where restoring simply inserts a new revision copying an old one's body — restore is itself just another save, so the log is never mutated or truncated by it.

Revisions are never pruned. Volume is inherently low (a personal About page, edited occasionally, each revision a few KB of text) — not worth the added complexity of a retention policy for a problem this scale will never hit, matching the "leave them" call already made for dev-image git tags (ADR-0008).

## Considered Options

- Skills folded into the markdown body via a custom convention — rejected: fragile, unprecedented in this codebase, for no real savings over a small table (see above).
- Categories as their own manageable entity (addable/renameable/reorderable, independent of Skills) — rejected for now as more surface than the stated need; noted as a possible future improvement if it turns out to matter, not built speculatively today.
- Version-counter-only (matching Posts/Projects) instead of real history — rejected: explicitly does not deliver "go back to previous states," which was the actual ask.
