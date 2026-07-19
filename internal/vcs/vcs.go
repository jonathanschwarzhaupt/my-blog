package vcs

import "runtime/debug"

// Version reports the running binary's version. Go's own module-version
// computation already does exactly what this needs: bi.Main.Version is the
// git tag when the build was made exactly at a tagged commit, otherwise a
// pseudo-version embedding the commit hash — with a "+dirty" suffix
// automatically appended if the working tree had uncommitted changes at
// build time. No extra reconstruction from bi.Settings is needed (see
// docs/adr/0007-fix-vcs-version-to-report-tags.md — this used to hand-roll
// that reconstruction from vcs.revision/vcs.modified, which is why it never
// showed a tag).
func Version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return bi.Main.Version
}
