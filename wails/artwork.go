package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Getting artwork back out of the project.
//
// An author who keeps a copy of their wrap inside the .draftline may one day
// have only that copy: the original gets moved, renamed, or lost with the
// machine it was made on. Unticking "keep a copy" would then destroy the
// artwork, and no amount of wording makes that acceptable.
//
// So the project can always hand a stored image back. It is the same act as an
// export — a save dialog and a file written where the author says — and it is
// what the Book & Editions screen does before it removes anything it cannot be
// sure exists elsewhere.

// ExportStoredArtwork writes one image held in the project to a file the
// author chooses. member is the archive member's own name, as the record
// spells it; suggested is the file name to offer.
func (a *App) ExportStoredArtwork(editionID, member, suggested string) (result types.ExportResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.ExportResult{Success: false, Error: fmt.Sprintf("that artwork could not be written: %v", r)}
		}
	}()

	editionID = strings.TrimSpace(editionID)
	member = strings.TrimSpace(member)
	if editionID == "" || member == "" {
		return types.ExportResult{Success: false, Error: "there is no copy of that artwork in this project."}
	}
	data, err := a.coverBytes(editionID, filepath.Base(member))
	if err != nil || len(data) == 0 {
		return types.ExportResult{Success: false, Error: "the copy of that artwork in this project could not be read."}
	}

	name := strings.TrimSpace(suggested)
	if name == "" {
		name = filepath.Base(member)
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save artwork",
		DefaultFilename: name,
		Filters: []runtime.FileFilter{
			{DisplayName: "Artwork (*" + filepath.Ext(name) + ")", Pattern: "*" + filepath.Ext(name)},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}
	// The same write every other export uses: built beside the destination and
	// renamed into place, so a failure leaves nothing half-written.
	if err := writeArtworkFile(path, data); err != nil {
		return types.ExportResult{Success: false, Error: "that file could not be written."}
	}
	return types.ExportResult{Success: true, FilePath: path}
}

func writeArtworkFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".draftline-art-*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	clean := func() {
		_ = temp.Close()
		_ = os.Remove(name)
	}
	if _, err := temp.Write(data); err != nil {
		clean()
		return err
	}
	if err := temp.Sync(); err != nil {
		clean()
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
