package readaloud

import (
	"fmt"
	"os"
	"path/filepath"
)

// Status describes the sole native Read Aloud bundle.
type Status struct {
	Installed        bool     `json:"installed"`
	Dir              string   `json:"dir"`
	BytesTotal       int64    `json:"bytes_total"`
	BytesOnDisk      int64    `json:"bytes_on_disk"`
	Missing          []string `json:"missing"`
	NativeSupported  bool     `json:"native_supported"`
	NativeInstalled  bool     `json:"native_installed"`
	NativeBytesTotal int64    `json:"native_bytes_total"`
}

// Check inspects dir against the manifest. Presence is judged by size only —
// a full hash of 130 MB on every settings-dialog open would be wasteful; the
// installer verifies hashes before files ever land here.
func Check(dir string) Status {
	status := Status{
		Dir: dir, BytesTotal: TotalBytes(), Missing: []string{},
		NativeSupported: NativeSupported(), NativeInstalled: NativeSupported(),
		NativeBytesTotal: NativeBytes(),
	}
	for _, art := range Manifest() {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(art.Name)))
		present := err == nil && info.Size() == art.Bytes
		if !present {
			status.Missing = append(status.Missing, art.Name)
			status.NativeInstalled = false
			continue
		}
		status.BytesOnDisk += info.Size()
	}
	if status.NativeInstalled {
		if info, err := os.Stat(filepath.Join(dir, "native", "model", "espeak-ng-data", "phontab")); err != nil || info.IsDir() {
			status.NativeInstalled = false
		}
	}
	status.Installed = status.NativeInstalled
	return status
}

// Remove deletes the downloaded bundle — every manifest file, the manifest
// itself, and the directory — and confirms nothing is left behind. As a
// guard against a misconstructed path ever reaching RemoveAll, it insists on
// the expected directory name.
func Remove(dir string) error {
	if filepath.Base(dir) != "kokoro" {
		return fmt.Errorf("refusing to remove unexpected directory %q", dir)
	}
	ShutdownNative()
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("model directory still present after removal: %s", dir)
	}
	return nil
}
