package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"draftline/internal/book"
	"draftline/internal/coverart"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// The print-ready wrap: the author's own wraparound artwork for one printed
// format. Draftline never makes one and never alters one. It reads what the
// file is, records where it lives, and keeps a copy inside the project only
// when asked — because a wrap is tens of megabytes and the whole project is
// re-sent every time it syncs.
//
// Like the cover, a wrap is chosen by PATH. The picker runs here and hands
// back a location; the bytes never cross the bridge.

// wrapSizeCap refuses a file no printer would have produced, before anything
// is read into memory.
const wrapSizeCap = 2 << 30 // 2 GB

// AttachWrapDialog asks for a file and attaches it to one format.
func (a *App) AttachWrapDialog(editionID, formatID string) types.WrapResult {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose print-ready wrap artwork",
		DefaultDirectory: a.getSettings().DefaultSaveDir,
		Filters: []runtime.FileFilter{
			{DisplayName: "Print artwork (*.pdf;*.tif;*.tiff;*.png;*.jpg;*.jpeg;*.psd;*.ai;*.eps)", Pattern: "*.pdf;*.tif;*.tiff;*.png;*.jpg;*.jpeg;*.psd;*.ai;*.eps"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	})
	if err != nil {
		return types.WrapResult{Error: err.Error()}
	}
	if path == "" {
		return types.WrapResult{Cancelled: true}
	}
	return a.AttachWrap(editionID, formatID, path)
}

// AttachWrap records the file at path as one format's wrap. It reads the
// file's own facts and does not copy it; keeping a copy is a separate,
// deliberate act (SetWrapStored), because it is what costs disk and sync.
func (a *App) AttachWrap(editionID, formatID, sourcePath string) (result types.WrapResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.WrapResult{Error: fmt.Sprintf("that artwork could not be read: %v", r)}
		}
	}()

	formatID = strings.TrimSpace(formatID)
	if formatID == "" {
		return types.WrapResult{Error: "choose a format to attach this artwork to"}
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return types.WrapResult{Error: "that file could not be opened"}
	}
	if info.IsDir() {
		return types.WrapResult{Error: "that is a folder, not artwork"}
	}
	if info.Size() > wrapSizeCap {
		return types.WrapResult{Error: "that file is larger than Draftline will take"}
	}

	wrap := &types.EditionWrap{
		FileName:    filepath.Base(sourcePath),
		Bytes:       info.Size(),
		Stored:      false,
		StoredLabel: byteLabel(info.Size()),
		SourcePath:  sourcePath,
		Attached:    time.Now().UTC().Format(time.RFC3339),
	}
	// Dimensions when the format states them. A PDF states none, and a wrap
	// with no pixel size is still a wrap.
	if w, h, ok := imageSize(sourcePath); ok {
		wrap.Width, wrap.Height = w, h
	}
	// A small copy for the screen, so looking at the panel never loads a
	// print-resolution file. A PDF or a PSD makes none — Go cannot rasterise
	// them — and the panel shows the file instead of a picture of it.
	if preview, ok := coverart.Preview(sourcePath); ok {
		name := "wrap-" + formatID + "-preview" + previewExt(preview.Encoding)
		// hold, not cache: the preview has to reach the project file, or
		// reopening the book shows a broken picture where the wrap was.
		a.covers.hold(editionID, coverFiles{name: preview.Data}, previewPrefix(formatID))
		wrap.PreviewFile = name
		wrap.PreviewWidth = preview.Width
		wrap.PreviewHeight = preview.Height
	}
	if sum, err := fileChecksum(sourcePath); err == nil {
		wrap.SourceChecksum = sum
	}
	return types.WrapResult{Success: true, Wrap: wrap}
}

// SetWrapStored copies the artwork into the project, or drops that copy and
// keeps only the record of where the file lives. It takes the wrap it is
// changing and hands back the changed one, the same way a cover does, so the
// screen holds the record and this holds the bytes.
func (a *App) SetWrapStored(editionID, formatID string, wrap types.EditionWrap, keep bool) (result types.WrapResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.WrapResult{Error: fmt.Sprintf("that copy could not be made: %v", r)}
		}
	}()

	editionID = strings.TrimSpace(editionID)
	formatID = strings.TrimSpace(formatID)
	if editionID == "" || formatID == "" {
		return types.WrapResult{Error: "choose a format first"}
	}

	changed := wrap
	member := wrapMember(editionID, formatID, wrap.FileName)

	if !keep {
		// Dropping the copy leaves the record: the file is still the wrap,
		// it just lives on disk now. An empty entry supersedes the stored
		// bytes on the next save.
		// The stored copy goes; the small preview stays, because the screen
		// still shows the artwork whether or not its bytes live here.
		a.covers.hold(editionID, coverFiles{}, storedWrapPrefix(formatID))
		changed.Stored = false
		changed.Member = ""
		return types.WrapResult{Success: true, Wrap: &changed}
	}

	if strings.TrimSpace(wrap.SourcePath) == "" {
		return types.WrapResult{Error: "Draftline no longer knows where that artwork is. Upload it again."}
	}
	data, err := os.ReadFile(wrap.SourcePath)
	if err != nil {
		return types.WrapResult{Error: "that file could not be read. It may have moved since it was attached."}
	}
	a.covers.hold(editionID, coverFiles{filepath.Base(member): data}, storedWrapPrefix(formatID))
	changed.Stored = true
	changed.Member = member
	changed.Bytes = int64(len(data))
	changed.StoredLabel = byteLabel(changed.Bytes)
	return types.WrapResult{Success: true, Wrap: &changed}
}

// byteLabel words a size the way the screen says it.
func byteLabel(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("about %.1f GB", float64(n)/float64(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("about %d MB", n/(1<<20))
	default:
		return fmt.Sprintf("about %d KB", max64(1, n/(1<<10)))
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// imageSize reads only the header of a format Go can decode. A PDF or a PSD
// returns nothing, which is not a failure.
func imageSize(path string) (int, int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// fileChecksum fingerprints the author's own file so a later export can tell
// whether it is still the artwork that was attached.
func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// previewExt names the small copy by what it actually is.
func previewExt(encoding string) string {
	if encoding == "png" {
		return ".png"
	}
	return ".jpg"
}

// The two edition-relative prefixes a wrap owns. They are kept apart so that
// dropping the stored copy leaves the preview alone: the screen goes on
// showing the artwork whether or not its bytes live in the project.
func storedWrapPrefix(formatID string) string { return "wrap-" + formatID + "." }

func previewPrefix(formatID string) string { return "wrap-" + formatID + "-preview." }

// wrapMember is where a stored wrap lives inside the archive.
func wrapMember(editionID, formatID, fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	return book.CoverPrefix(editionID) + "wrap-" + formatID + ext
}
