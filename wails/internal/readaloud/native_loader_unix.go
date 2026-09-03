//go:build darwin || linux

package readaloud

import "github.com/ebitengine/purego"

func nativeLibraryNames() (string, string) {
	if runtimeLibrarySuffix == "dylib" {
		return "libonnxruntime.dylib", "libsherpa-onnx-c-api.dylib"
	}
	return "libonnxruntime.so", "libsherpa-onnx-c-api.so"
}

func openNativeLibrary(path string, dependency bool) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func closeNativeLibrary(handle uintptr) error { return purego.Dlclose(handle) }
