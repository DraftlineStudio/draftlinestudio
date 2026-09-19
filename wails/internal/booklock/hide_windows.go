package booklock

import (
	"path/filepath"
	"syscall"
)

// hide marks the sidecar hidden.
//
// The name starts with a dot, which hides it on macOS and Linux and does
// nothing at all on Windows — where an author would otherwise find a
// .novel.draftline.lock sitting beside every book they have open, in the
// folder they keep their manuscripts in. Word does the same thing to its own
// owner files (~$Filename.docx) for the same reason.
//
// Best effort. A sidecar that could not be hidden is untidy; failing the open
// over it would be absurd.
func hide(path string) {
	name, err := syscall.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return
	}
	attrs, err := syscall.GetFileAttributes(name)
	if err != nil {
		return
	}
	_ = syscall.SetFileAttributes(name, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
}
