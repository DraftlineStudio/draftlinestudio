package plugins

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	stubOnce  sync.Once
	stubPath  string
	stubError error
)

// buildStubSidecar compiles testdata/stubsidecar once per test run and copies
// the executable into a fresh plugin directory for the calling test.
func buildStubSidecar(t *testing.T) string {
	t.Helper()
	stubOnce.Do(func() {
		dir, err := os.MkdirTemp("", "draftline-stub-sidecar")
		if err != nil {
			stubError = err
			return
		}
		name := "stubsidecar"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		stubPath = filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", stubPath, "./testdata/stubsidecar")
		out, err := cmd.CombinedOutput()
		if err != nil {
			stubError = err
			stubPath = ""
			_ = os.WriteFile(filepath.Join(dir, "build.log"), out, 0o644)
		}
	})
	if stubError != nil || stubPath == "" {
		t.Skipf("cannot build stub sidecar (go toolchain unavailable?): %v", stubError)
	}
	pluginDir := t.TempDir()
	dst := filepath.Join(pluginDir, filepath.Base(stubPath))
	if err := copyFile(stubPath, dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
