package types

// Cover art on an edition.
//
// This file holds the RECORD of a cover, never the cover itself. The bytes of
// an image are the one thing that must not join BookData: the whole struct is
// serialised to JSON and sent across the Wails bridge on every save, and an
// autosave runs five seconds after a keystroke. A megabyte of JPEG on this
// struct would be a megabyte base64-encoded through the webview every time the
// author paused for breath. The images live in the archive under
// editions/<edition id>/ and, while the app is running, in a cache beside the
// app that is keyed by edition and never crosses the bridge.
//
// The cover belongs to the EDITION, not to the book. That is the design brief
// and it is also simply true: the art of a first edition stays with the first
// edition's ISBNs when a second edition is reset with new art.

// EditionCover is everything the screen needs to describe an attached cover,
// and everything a later export needs to find the artwork it was made from.
type EditionCover struct {
	// ID changes every time a cover is attached. It is what a display URL
	// carries so that replacing the art actually changes the picture on
	// screen, rather than showing whatever the webview cached under the
	// unchanged address editions/<edition id>/cover_thumb.jpg.
	ID string `json:"id"`

	// File and ThumbFile are member names inside editions/<edition id>/.
	// The extension is not always .jpg: flat artwork is kept as a PNG when
	// PNG is smaller, which it is by a wide margin on a two-colour cover.
	File      string `json:"file"`
	ThumbFile string `json:"thumb_file"`
	// LargeFile is the opt-in 2400 x 3840 derivative, empty unless asked for.
	LargeFile string `json:"large_file,omitempty"`

	Width       int `json:"width"`
	Height      int `json:"height"`
	Bytes       int `json:"bytes"`
	ThumbWidth  int `json:"thumb_width"`
	ThumbHeight int `json:"thumb_height"`
	ThumbBytes  int `json:"thumb_bytes"`
	LargeWidth  int `json:"large_width,omitempty"`
	LargeHeight int `json:"large_height,omitempty"`
	LargeBytes  int `json:"large_bytes,omitempty"`

	// Encoding is "jpeg" or "png"; Quality is the JPEG quality the search
	// settled on, and 0 for a PNG.
	Encoding string `json:"encoding"`
	Quality  int    `json:"quality"`

	// What was done to the artwork, and has to be visible on screen.
	Greyscale         bool `json:"greyscale,omitempty"`
	ConvertedFromCMYK bool `json:"converted_from_cmyk,omitempty"`
	FlattenedAlpha    bool `json:"flattened_alpha,omitempty"`

	// The print-ready original, recorded rather than copied. A print export
	// reads it from here at export time; if it has moved, the screen says so
	// and asks rather than quietly printing the 1600-pixel derivative at trim
	// size.
	SourcePath     string `json:"source_path,omitempty"`
	SourceChecksum string `json:"source_checksum,omitempty"`
	SourceBytes    int64  `json:"source_bytes,omitempty"`
	SourceWidth    int    `json:"source_width,omitempty"`
	SourceHeight   int    `json:"source_height,omitempty"`
	SourceModified string `json:"source_modified,omitempty"`
	SourceFormat   string `json:"source_format,omitempty"`

	// Attached is when this cover was made, RFC 3339.
	Attached string `json:"attached,omitempty"`

	// Notes are the sentences the panel shows about this particular
	// conversion: naive CMYK, flattened transparency, an unusual shape.
	Notes []string `json:"notes,omitempty"`
}

// CoverResult is what attaching a cover returns to the frontend. It carries
// the record and never the image.
type CoverResult struct {
	Success bool          `json:"success"`
	Error   string        `json:"error,omitempty"`
	Cover   *EditionCover `json:"cover,omitempty"`
	// Cancelled distinguishes a dismissed file dialog from a failure, so the
	// screen can say nothing at all rather than show an error.
	Cancelled bool `json:"cancelled,omitempty"`
}

// CoverSourceReport answers "is the print-ready original still there".
type CoverSourceReport struct {
	Status  string `json:"status"` // "present" | "moved" | "changed"
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}
