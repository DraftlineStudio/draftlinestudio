package readaloud

import (
	"fmt"
	"os"
	"path/filepath"
)

// Status describes the on-disk state of the Read Aloud bundle. Installed,
// BytesTotal, and Missing describe the mandatory core (CPU) bundle; the
// optional GPU model is reported separately.
type Status struct {
	Installed     bool     `json:"installed"`
	Dir           string   `json:"dir"`
	BytesTotal    int64    `json:"bytes_total"`
	BytesOnDisk   int64    `json:"bytes_on_disk"`
	Missing       []string `json:"missing"`
	GPUInstalled  bool     `json:"gpu_installed"`
	GPUBytesTotal int64    `json:"gpu_bytes_total"`
}

// Check inspects dir against the manifest. Presence is judged by size only —
// a full hash of 130 MB on every settings-dialog open would be wasteful; the
// installer verifies hashes before files ever land here.
func Check(dir string) Status {
	status := Status{
		Dir: dir, BytesTotal: TotalBytes(), Missing: []string{},
		GPUInstalled: true, GPUBytesTotal: groupBytes(GroupGPU),
	}
	for _, art := range Manifest() {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(art.Name)))
		present := err == nil && info.Size() == art.Bytes
		if art.Group == GroupGPU {
			if !present {
				status.GPUInstalled = false
			} else {
				status.BytesOnDisk += info.Size()
			}
			continue
		}
		if !present {
			status.Missing = append(status.Missing, art.Name)
			continue
		}
		status.BytesOnDisk += info.Size()
	}
	status.Installed = len(status.Missing) == 0
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
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("model directory still present after removal: %s", dir)
	}
	return nil
}
