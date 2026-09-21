//go:build !windows && !darwin

package booklock

// Linux and the BSDs hide by a leading dot, which sync clients refuse to
// upload, so the sidecar stays visible there. A visible lock beats one that
// never reaches the other device.
func hide(string) {}
