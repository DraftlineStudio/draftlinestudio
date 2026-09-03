//go:build !windows && !darwin && !linux

package readaloud

import "fmt"

func nativeLibraryNames() (string, string) { return "", "" }
func openNativeLibrary(string, bool) (uintptr, error) {
	return 0, fmt.Errorf("native read aloud is unsupported on this platform")
}
func closeNativeLibrary(uintptr) error { return nil }
