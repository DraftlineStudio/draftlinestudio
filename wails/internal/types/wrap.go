package types

// The print-ready wrap: the full wraparound a printer needs, which is back,
// spine and front in one piece at print resolution with bleed.
//
// Draftline never makes one and never alters one. It records what the file is
// and where it lives, and keeps a copy inside the project only when asked,
// because a wrap is tens of megabytes and the whole project is re-sent every
// time it syncs.

// EditionWrap is one printed format's wraparound artwork.
type EditionWrap struct {
	// FileName is the author's own name for the file, shown as they know it.
	FileName string `json:"file_name"`
	Bytes    int64  `json:"bytes,omitempty"`
	// Width and Height are pixels, when the file states them. A PDF or an
	// unreadable format states none, and the record is still valid.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// SizeLabel is the physical size a printer works in, when one can be
	// stated ("12.25 × 9.25 in · 300 dpi").
	SizeLabel string `json:"size_label,omitempty"`
	// Stored is true when the bytes are inside the project. False means only
	// SourcePath is known, and the file must still be there at export.
	Stored bool `json:"stored"`
	// StoredLabel is what keeping the copy costs, already worded.
	StoredLabel string `json:"stored_label,omitempty"`
	// Member is the archive member holding the bytes when Stored.
	Member string `json:"member,omitempty"`
	// SourcePath is always recorded, stored or not: it is the route back to
	// the author's own working file.
	SourcePath string `json:"source_path,omitempty"`
	// SourceChecksum detects the file being replaced or moved since it was
	// attached, so an export can say so instead of shipping the wrong art.
	SourceChecksum string `json:"source_checksum,omitempty"`
	Attached       string `json:"attached,omitempty"`
	// PreviewFile is a small copy the screen shows so a panel never loads a
	// print-resolution file. Empty for a format Go cannot decode, such as a
	// PDF or a PSD, which are ordinary wrap formats.
	PreviewFile   string `json:"preview_file,omitempty"`
	PreviewWidth  int    `json:"preview_width,omitempty"`
	PreviewHeight int    `json:"preview_height,omitempty"`
}

// WrapResult is the answer to attaching or changing a wrap.
type WrapResult struct {
	Success bool         `json:"success"`
	Error   string       `json:"error,omitempty"`
	Wrap    *EditionWrap `json:"wrap,omitempty"`
	// Cancelled distinguishes a dismissed file dialog from a failure, so the
	// screen says nothing at all rather than showing an error.
	Cancelled bool `json:"cancelled,omitempty"`
}
