# Project ordering via an editable float order_key, plus user-editable created_at

The `/projects` list needs a curator-controlled default order, with a secondary option to view by creation date. We add an `order_key double precision` column to `projects` — a binary float, not `numeric` (Postgres's arbitrary-precision decimal): sqlc/pgx maps `numeric` to `pgtype.Numeric`, a much more awkward type to parse/format/do arithmetic on in Go than a plain `float64`, and this column has no need for `numeric`'s exactness (it's a relative-position key, never summed or compared for equality against a business value). New projects default to `max(order_key) + 1`, and repositioning (including moving something to the front) is done by typing any number — decimals allowed — between the two neighbors you want it to land between, or below the current minimum to move to the front. No uniqueness constraint; a tie-break by `created_at` (or `id`) keeps ordering deterministic if two rows ever collide.

This is the same idea as fractional indexing (Notion, Linear, Trello's "LexoRank"), fitted to this codebase's plain server-rendered admin forms instead of a drag-and-drop UI: a float column lets any insertion point be expressed as a single number without renumbering existing rows. A full drag-and-drop reorder endpoint (Considered Options) was rejected as new UI/JS surface disproportionate to a "few in number by design" collection edited by one person.

Separately, `created_at` becomes directly user-editable (a form field, not just an internal timestamp), repurposed as an optional secondary sort (asc/desc, public toggle on `/projects`) rather than a pure audit trail. This is a deliberate trade — `created_at` no longer reliably means "when the row was created" — accepted because a single dedicated `order_key` plus a separately-named immutable timestamp was judged not worth a second column for a field only the site owner ever edits.

## Considered Options

- Separate immutable `created_at` (audit) + new mutable `sort_date` column — rejected as an unnecessary second date column for a single-operator admin feature.
- Full drag-and-drop reordering (JS drop handler + backend renumbering endpoint) — rejected as disproportionate new UI surface for this app's current all-server-rendered-forms admin pattern; can be layered on top of the same `order_key` column later without another migration.
