package readaloud

import (
	"fmt"
	"os"
	"path/filepath"
)

// Status describes the on-disk state of the Read Aloud bundle.
type Status struct {
	Installed   bool     `json:"installed"`
	Dir         string   `json:"dir"`
	BytesTotal  int64    `json:"bytes_total"`
	BytesOnDisk int64    `json:"bytes_on_disk"`
	Missing     []string `json:"missing"`
}

// Check inspects dir against the manifest. Presence is judged by size only —
// a full hash of 130 MB on every settings-dialog open would be wasteful; the
// installer verifies hashes before files ever land here.
func Check(dir string) Status {
	status := Status{Dir: dir, BytesTotal: TotalBytes(), Missing: []string{}}
	for _, art := range Manifest() {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(art.Name)))
		if err != nil || info.Size() != art.Bytes {
			status.Missing = append(status.Missing, art.Name)
			continue
		}
		status.BytesOnDisk += info.Size()
	}
	status.Installed = len(status.Missing) == 0
	return status
}

// Remove deletes the downloaded bundle. As a guard against a misconstructed
// path ever reaching RemoveAll, it insists on the expected directory name.
func Remove(dir string) error {
	if filepath.Base(dir) != "kokoro" {
		return fmt.Errorf("refusing to remove unexpected directory %q", dir)
	}
	return os.RemoveAll(dir)
}
