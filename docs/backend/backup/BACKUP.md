# Backup Package

`internal/backup/` handles automatic backup creation before saves.

## Functions

### Create(path string) error
Creates a numbered backup of the file at `path` before it's overwritten. Backups are stored in a `.backups/` subdirectory next to the original file.

**Behavior:**
- Skips if source file doesn't exist (new file)
- Creates `.backups/` directory if needed
- Rotates old backups (keeps last 10 by default)
- Names backups as `filename.001.draftline`, `filename.002.draftline`, etc.

### List(path string) ([]types.BackupInfo, error)
Returns a list of available backups for the given file path, sorted by modification time (newest first).

### Restore(backupPath, targetPath string) error
Restores a backup file to the target location.

## Configuration

- `maxBackups = 10` - Maximum number of backups to retain per file

## Usage

Called automatically by `book.Write()` before saving a project file.
