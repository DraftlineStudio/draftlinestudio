package book

import "fmt"

// requireArchiveVersion refuses archived data written by a version of the
// format this build does not know. It is the gate planner.json already applies
// to itself, in a form the members added under editions/ can share.
//
// The rule is deliberately narrow: an unrecognised version is an error. The
// file is not coerced into the shape this build expects and is not rewritten,
// because a save rebuilds the archive and would otherwise overwrite data a
// newer Draftline wrote with a lossy reading of it. The manifest does not do
// this, which is why a manifest from the future is silently downgraded; new
// archive members do not repeat that.
func requireArchiveVersion(name string, got, want int) error {
	if got != want {
		return fmt.Errorf("%s version %d is not supported (this version of Draftline reads version %d)", name, got, want)
	}
	return nil
}
