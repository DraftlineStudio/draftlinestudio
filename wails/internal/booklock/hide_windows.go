package booklock

import (
	"path/filepath"
	"syscall"
)

// hide sets the hidden attribute, which is how the sidecar stays out of the
// way now that its name cannot start with a dot. Best effort: an unhidden
// sidecar is untidy but choices are few so it is what it is.
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
