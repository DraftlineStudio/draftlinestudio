package coverart

// The print-ready original, which Draftline does not copy.
//
// The file an author attaches is usually the good one: a 300 dpi wrap, a
// layered export, forty megabytes of it. What goes into the .draftline is a
// 1600-pixel front cover, and that is deliberate - the project file is opened,
// autosaved, backed up and carried around, and none of those should cost forty
// megabytes a time.
//
// So the original is recorded rather than kept: where it was, how big it was,
// and a checksum of its contents. That is enough to answer the only question
// that matters later, at export time, when a print export wants the artwork at
// print size: is the file still the one the cover was made from?
//
// Three answers are possible and they are not the same answer:
//
//	Present   the file is there and its checksum matches. Use it.
//	Moved     nothing is at that path any more. ASK - do not quietly export
//	          the 1600-pixel derivative at print size, which would print a
//	          cover at about 190 dpi and look exactly like a mistake.
//	Changed   a file is there and its contents are not the ones the cover was
//	          made from. Say so; it may be a corrected version, and it may be
//	          a different book.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Fingerprint is what is recorded about the file an author attached.
type Fingerprint struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Bytes    int64  `json:"bytes"`
	// Modified is the file's own timestamp when it was attached, kept for the
	// screen rather than for the comparison: the checksum decides.
	Modified string `json:"modified"`
}

// Source status values. They are strings because they cross to the frontend
// and a number would mean nothing in a JSON file somebody opens in five years.
const (
	SourcePresent = "present"
	SourceMoved   = "moved"
	SourceChanged = "changed"
)

// SourceReport is the answer to "is the original still there", in a form a
// panel can show without composing a sentence of its own.
type SourceReport struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

// TakeFingerprint hashes the file at path.
func TakeFingerprint(path string) (Fingerprint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Fingerprint{}, err
	}
	sum, err := checksum(path)
	if err != nil {
		return Fingerprint{}, err
	}
	return Fingerprint{
		Path:     path,
		Checksum: sum,
		Bytes:    info.Size(),
		Modified: info.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

// CheckSource compares a recorded fingerprint against the disk now.
//
// A fingerprint with no path at all reports Moved with a message saying so
// plainly, which is the truthful answer for a cover attached by a build that
// did not record one.
func CheckSource(print Fingerprint) SourceReport {
	if print.Path == "" {
		return SourceReport{
			Status:  SourceMoved,
			Message: "Draftline did not record where this artwork came from, so it cannot find the print-ready original. Attach the cover again from the original file if you want a print export to use it.",
		}
	}
	base := filepath.Base(print.Path)
	info, err := os.Stat(print.Path)
	if err != nil || info.IsDir() {
		return SourceReport{
			Status: SourceMoved,
			Path:   print.Path,
			Message: fmt.Sprintf(
				"The artwork this cover was made from is no longer at %s. Draftline keeps the 1600-pixel ebook cover in the project, not the print-ready original, so a print export cannot use the full-size artwork until you point Draftline at it again. Printing the ebook cover at trim size would come out at about 190 dots per inch, and it would look like it.",
				print.Path),
		}
	}
	sum, err := checksum(print.Path)
	if err != nil {
		return SourceReport{
			Status:  SourceMoved,
			Path:    print.Path,
			Message: fmt.Sprintf("%s is there but could not be read, so Draftline cannot tell whether it is still the artwork this cover was made from.", base),
		}
	}
	if sum != print.Checksum {
		return SourceReport{
			Status: SourceChanged,
			Path:   print.Path,
			Message: fmt.Sprintf(
				"%s is still there, but it is not the file this cover was made from any more. If that is a corrected version, attach it again so the cover and the original agree.",
				base),
		}
	}
	return SourceReport{
		Status:  SourcePresent,
		Path:    print.Path,
		Message: fmt.Sprintf("%s is where it was when the cover was made, unchanged.", base),
	}
}

func checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
