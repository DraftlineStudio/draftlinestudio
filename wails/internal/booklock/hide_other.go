//go:build !windows

package booklock

// hide is a no-op everywhere but Windows: the sidecar's name begins with a
// dot, which is all macOS and Linux need. See hide_windows.go.
//
// The build constraint is the load-bearing line in this file. The name says
// "other", which is not a platform, so without it this would compile on
// Windows too and collide with the real implementation.
func hide(string) {}
