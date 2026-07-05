package layout

// FeatureFlags gates which optional nav sections the shared layout renders,
// and which routes cmd/blog's routes() registers (see ADR-0003) — set once
// from the parsed -features flag (e.g. -features=admin) in main().
//
// Features must be set once at startup, before the server begins accepting
// requests, and never mutated afterward — it's read concurrently by every
// request rendering the shared layout.
type FeatureFlags struct {
	// Admin shows nav links to the write routes (New Post, New Project) and,
	// in cmd/blog/routes.go, controls whether those routes are registered
	// at all. Since the public routes (Home, Projects, About, feed) are
	// always registered regardless of mode, admin mode's nav links to them
	// are real links, not dead ones.
	Admin bool
}

var Features FeatureFlags
