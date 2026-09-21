package booklock

import "syscall"

// UF_HIDDEN keeps the sidecar out of Finder without a dot in the name, which
// OneDrive and MEGA refuse to sync.
const ufHidden = 0x8000

func hide(path string) {
	_ = syscall.Chflags(path, ufHidden)
}
