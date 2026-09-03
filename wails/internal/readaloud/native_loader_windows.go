//go:build windows

package readaloud

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func nativeLibraryNames() (string, string) {
	return "onnxruntime.dll", "sherpa-onnx-c-api.dll"
}

func openNativeLibrary(path string, dependency bool) (uintptr, error) {
	// Loading ONNX Runtime first by absolute path lets the subsequent sherpa
	// DLL resolve its import without modifying process-wide PATH state.
	h, err := windows.LoadLibrary(filepath.Clean(path))
	return uintptr(h), err
}

func closeNativeLibrary(handle uintptr) error {
	return windows.FreeLibrary(windows.Handle(handle))
}
