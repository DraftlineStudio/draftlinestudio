package types

// What an edition export asks for.
//
// Exporting an edition is one act with one result: an archive holding every
// registered format that was chosen, each in its own folder with its own
// artwork. The wizard therefore sends the whole request at once rather than
// calling an exporter per format, because the alternative — several save
// dialogs and several files an author then has to gather — is the thing this
// replaces.
//
// Each item carries the full set of the wizard's answers rather than only the
// ones its own exporter reads. The wizard holds them as one object per format,
// and splitting them here would put the same knowledge in two places.

// BundleItem is one registered format going into the bundle.
type BundleItem struct {
	FormatID string `json:"format_id"`
	// Output is what Draftline writes for this format: epub, docx, pdf,
	// print-pdf, hc or audio. A hardcover interior is the print exporter and
	// an audiobook script is the reading-copy exporter, so this is not the
	// same thing as the exporter that runs.
	Output string          `json:"output"`
	Shared ExportOptions   `json:"shared"`
	EPUB   EPUBOptions     `json:"epub"`
	PDF    PDFOptions      `json:"pdf"`
	Audio  AudioOptions    `json:"audio"`
	Print  PrintPDFOptions `json:"print"`
}

// BundleRequest is the whole edition export.
type BundleRequest struct {
	EditionID string `json:"edition_id"`
	// IncludeArtwork carries the cover and the print-ready wrap beside each
	// interior. Off, the bundle holds interiors only.
	IncludeArtwork bool         `json:"include_artwork"`
	Items          []BundleItem `json:"items"`
}
