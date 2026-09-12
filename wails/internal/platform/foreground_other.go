//go:build !windows

package platform

// FocusProcessWindow is Windows-only; elsewhere the caller's message ("this
// book is already open in another window") stands on its own.
func FocusProcessWindow(pid int) bool {
	return false
}
