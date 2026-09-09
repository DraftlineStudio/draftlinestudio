package export

import (
	"os"
	"path/filepath"
)

// writeExportFile keeps the destination untouched until the entire edition
// has been generated and durably written beside it.
func writeExportFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".draftline-export-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	clean := func() { _ = os.Remove(tempName) }
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		clean()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		clean()
		return err
	}
	if err := temp.Close(); err != nil {
		clean()
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		clean()
		return err
	}
	return nil
}
