// Package fsutil provides filesystem helpers shared across the backend.
package fsutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// renameAttempts covers transient sharing violations on Windows (antivirus,
// sync clients briefly holding the target open).
const renameAttempts = 3

// WriteFileAtomic writes data to path so that the destination always contains
// either its previous contents or the complete new contents, never a partial
// write. It writes to a temp file in the same directory (same volume, so the
// final rename is atomic), fsyncs, then renames over the target.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".draftline-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	// CreateTemp creates 0600; apply the requested permissions before the
	// file becomes visible at its final path. Best-effort on Windows.
	if err := os.Chmod(tmpName, perm); err != nil && !errors.Is(err, os.ErrPermission) {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	var renameErr error
	for attempt := 0; attempt < renameAttempts; attempt++ {
		if renameErr = os.Rename(tmpName, path); renameErr == nil {
			return nil
		}
		time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
	}
	cleanup()
	return fmt.Errorf("replace %s: %w", path, renameErr)
}
