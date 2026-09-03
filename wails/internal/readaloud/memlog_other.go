//go:build !windows

package readaloud

// MemorySnapshot is Windows-only for now; the leak under investigation is a
// WebView2 behavior. Other platforms simply produce no RSS lines.
func MemorySnapshot() []string {
	return nil
}
