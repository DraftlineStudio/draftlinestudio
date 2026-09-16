package export

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"draftline/internal/types"
)

// The distribution bundle.
//
// Exporting one edition produces one archive, not a handful of files scattered
// through a downloads folder. Inside it is a single folder named for the book
// and the edition, and inside that one folder per registered format holding
// everything that format needs to reach a printer or a storefront: the
// interior, and the artwork that goes on it.
//
//	Wide Water — First edition.zip
//	└── Wide Water — First edition/
//	    ├── Paperback/   interior.pdf  +  the print-ready wrap
//	    ├── Hardcover/   interior.pdf  +  its own wrap, at its own spine
//	    ├── eBook/       the EPUB      +  the storefront cover
//	    └── Audiobook/   the narrator script + the cover it opens on
//
// An export that belongs to no edition is still written as loose files. This
// shape exists because an edition is a thing being published, and the point of
// it is that nothing has to be gathered by hand afterwards.

// BundleMember is one file inside the bundle. It carries either the bytes of
// something just rendered or the location of something already on disk —
// print artwork can be hundreds of megabytes, and a file that already exists
// is copied through rather than read into memory first.
type BundleMember struct {
	Path   string
	Data   []byte
	Source string
}

// alreadyCompressed names the members that gain nothing from deflate. A JPEG
// or an EPUB recompressed is the same size and costs the time.
var alreadyCompressed = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".epub": true, ".zip": true, ".docx": true,
}

// Bundle writes the members to path as one archive. Nothing reaches the
// destination until the whole bundle is built, the same rule every other
// export follows: a half-written publication package is worse than none.
func Bundle(target string, members []BundleMember) types.ExportResult {
	if len(members) == 0 {
		return types.ExportResult{Success: false, Error: "there is nothing to put in this bundle"}
	}
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".draftline-bundle-*.tmp")
	if err != nil {
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	tempName := temp.Name()
	clean := func() {
		_ = temp.Close()
		_ = os.Remove(tempName)
	}

	zw := zip.NewWriter(temp)
	for _, member := range members {
		if err := writeBundleMember(zw, member); err != nil {
			clean()
			return types.ExportResult{Success: false, Error: err.Error()}
		}
	}
	if err := zw.Close(); err != nil {
		clean()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to finish the bundle: %v", err)}
	}
	if err := temp.Sync(); err != nil {
		clean()
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempName)
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	if err := os.Rename(tempName, target); err != nil {
		_ = os.Remove(tempName)
		return types.ExportResult{Success: false, Error: fmt.Sprintf("failed to write file: %v", err)}
	}
	return types.ExportResult{Success: true, FilePath: target}
}

func writeBundleMember(zw *zip.Writer, member BundleMember) error {
	method := zip.Deflate
	if alreadyCompressed[strings.ToLower(path.Ext(member.Path))] {
		method = zip.Store
	}
	w, err := zw.CreateHeader(&zip.FileHeader{Name: member.Path, Method: method})
	if err != nil {
		return fmt.Errorf("failed to add %s: %w", path.Base(member.Path), err)
	}
	if member.Source != "" {
		f, err := os.Open(member.Source)
		if err != nil {
			return fmt.Errorf("%s could not be read. It may have moved since it was attached.", filepath.Base(member.Source))
		}
		defer func() { _ = f.Close() }()
		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to add %s: %w", filepath.Base(member.Source), err)
		}
		return nil
	}
	if _, err := w.Write(member.Data); err != nil {
		return fmt.Errorf("failed to add %s: %w", path.Base(member.Path), err)
	}
	return nil
}

// SafeName makes one path segment out of whatever an author called their book
// or their edition. It is not a general sanitiser: it exists so that a title
// with a colon in it does not produce an archive Windows refuses to unpack.
func SafeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Untitled"
	}
	var out strings.Builder
	for _, r := range trimmed {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			out.WriteRune('-')
		default:
			if r < 0x20 {
				out.WriteRune(' ')
				continue
			}
			out.WriteRune(r)
		}
	}
	// A trailing dot or space is legal in a zip and not on Windows.
	cleaned := strings.TrimRight(out.String(), ". ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if cleaned == "" {
		return "Untitled"
	}
	if len(cleaned) > 120 {
		cleaned = strings.TrimSpace(cleaned[:120])
	}
	return cleaned
}

// UniqueFolder keeps two formats with the same printed word — two paperbacks,
// say — in two folders rather than merging them into one.
func UniqueFolder(taken map[string]bool, name string) string {
	folder := SafeName(name)
	if !taken[strings.ToLower(folder)] {
		taken[strings.ToLower(folder)] = true
		return folder
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s (%d)", folder, n)
		if !taken[strings.ToLower(candidate)] {
			taken[strings.ToLower(candidate)] = true
			return candidate
		}
	}
}
