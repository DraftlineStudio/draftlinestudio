# Backup Package

`internal/backup/` handles automatic backup creation before saves.

## Functions

### Create(filePath string) error
Creates a numbered backup of the file at `filePath` before it's overwritten. Backups are stored under the OS config directory: `os.UserConfigDir()/draftline/backups/<sha256-prefix-of-path>/`, created owner-only (0700) since backups can contain the full manuscript.

**Behavior:**
- Skips if source file doesn't exist (new file)
- Creates the backup directory if needed
- Rotates old backups: `backup.5.draftline` deleted, `backup.4` → `backup.5`, etc.
- Writes the current file to `backup.1.draftline`, plus an `info.json` recording the original path

### List(filePath string) []types.BackupInfo
Returns available backups for the given file path (backup number, path, modified time, size). No error return — an empty slice means no backups.

### Restore(filePath string, number int) types.SaveResult
Restores a backup by number (1 = most recent, 5 = oldest) over the current file, backing up the current state first so the restore is reversible.

## Configuration

- `MaxBackups = 5` - Maximum number of backups to retain per file

## Usage

Called automatically by `book.Write()` before saving a project file.
