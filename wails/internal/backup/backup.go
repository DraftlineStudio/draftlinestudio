// Package backup provides rolling backup functionality for Draftline projects.
package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"draftline/internal/types"
)

// MaxBackups is the maximum number of backups to keep per project.
const MaxBackups = 5

// Dir returns the backup directory for a given file path.
// Creates a subdirectory based on a hash of the file path.
func Dir(filePath string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	// Create hash of the file path for unique folder name
	hash := sha256.Sum256([]byte(filepath.Clean(filePath)))
	hashStr := hex.EncodeToString(hash[:8]) // First 8 bytes = 16 hex chars
	dir := filepath.Join(configDir, "draftline", "backups", hashStr)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// Create copies the current file to the backup directory before saving.
// Rotates existing backups: backup.5 deleted, backup.4 -> backup.5, etc.
func Create(filePath string) error {
	// Only backup if the file exists (skip for new files)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	backupDir := Dir(filePath)

	// Rotate existing backups (delete oldest, shift others)
	for i := MaxBackups; i >= 1; i-- {
		old := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i))
		if i == MaxBackups {
			// Delete the oldest backup
			_ = os.Remove(old)
		} else {
			// Rename backup.N to backup.N+1
			newName := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i+1))
			_ = os.Rename(old, newName)
		}
	}

	// Copy current file to backup.1
	src, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "backup.1.draftline")
	if err := os.WriteFile(backupPath, src, 0644); err != nil {
		return err
	}

	// Also write a small metadata file so we know what this backup is
	metaPath := filepath.Join(backupDir, "info.json")
	meta := map[string]string{
		"original_path": filePath,
		"last_backup":   time.Now().Format(time.RFC3339),
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaPath, metaBytes, 0644)

	return nil
}

// List returns available backups for the given file path.
func List(filePath string) []types.BackupInfo {
	if filePath == "" {
		return []types.BackupInfo{}
	}

	backupDir := Dir(filePath)
	var backups []types.BackupInfo

	for i := 1; i <= MaxBackups; i++ {
		path := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", i))
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		backups = append(backups, types.BackupInfo{
			Number:   i,
			Path:     path,
			Modified: info.ModTime().Format(time.RFC3339),
			Size:     info.Size(),
		})
	}

	return backups
}

// Restore restores a backup by number (1 = most recent, 5 = oldest).
// Creates a backup of the current state before restoring.
func Restore(filePath string, number int) types.SaveResult {
	if filePath == "" {
		return types.SaveResult{Success: false, Error: "no file currently open"}
	}
	if number < 1 || number > MaxBackups {
		return types.SaveResult{Success: false, Error: "invalid backup number"}
	}

	backupDir := Dir(filePath)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("backup.%d.draftline", number))

	// Check backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return types.SaveResult{Success: false, Error: "backup not found"}
	}

	// Read backup
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	// Before restoring, backup the current state (so restore is reversible)
	_ = Create(filePath)

	// Write backup data to current file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return types.SaveResult{Success: false, Error: err.Error()}
	}

	return types.SaveResult{Success: true, FilePath: filePath}
}
