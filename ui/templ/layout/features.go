package layout

// FeatureFlags gates which optional nav sections the shared layout renders.
// This anticipates blog and blog-admin eventually merging into one binary
// controlled by a -features flag (e.g. -features=admin,maybe-another-feature)
// instead of being two separate processes — the conditional is already in
// place in base.templ, so wiring a real flag later only means setting these
// fields from parsed CLI input instead of hardcoding them here.
//
// Features must be set once at startup, before the server begins accepting
// requests, and never mutated afterward — it's read concurrently by every
// request rendering the shared layout.
type FeatureFlags struct {
	// Admin shows nav links to blog-admin's write routes (New Post, New
	// Project). blog leaves this false since it doesn't serve those routes.
	// blog-admin sets it true. Today that means blog-admin's nav also shows
	// the public Home/Projects/About links even though blog-admin doesn't
	// serve those either — accepted as fine for now, since the merged
	// single-binary future is exactly the case where both would be real.
	Admin bool
}

var Features FeatureFlags
