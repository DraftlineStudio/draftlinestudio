package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"draftline/internal/export"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Exporting an edition.
//
// A reading copy or a from-scratch export is a file, and Draftline writes it
// the way it always has: one save dialog, one file. An edition is not a file.
// It is several objects that go out together — a paperback interior and the
// wrap that prints around it, an ebook and the cover a storefront shows, a
// script and the art it opens on — and the whole point of registering it is
// that those objects belong to each other.
//
// So one edition export is one archive, laid out so nothing has to be gathered
// afterwards:
//
//	Wide Water — First edition.zip
//	└── Wide Water — First edition/
//	    ├── Paperback/
//	    ├── Hardcover/
//	    ├── eBook/
//	    └── Audiobook/
//
// Every interior is rendered before anything is written, and the archive is
// built beside its destination and renamed into place, so a failure half way
// through leaves no half-published bundle behind.

// ExportEditionBundle writes one edition as a publication-ready archive.
func (a *App) ExportEditionBundle(b types.BookData, request types.BundleRequest) (result types.ExportResult) {
	defer func() {
		if r := recover(); r != nil {
			result = types.ExportResult{Success: false, Error: fmt.Sprintf("that bundle could not be built: %v", r)}
		}
	}()

	if b.Editions == nil {
		return types.ExportResult{Success: false, Error: "this book has no registered editions"}
	}
	edition, ok := b.Editions.FindEdition(strings.TrimSpace(request.EditionID))
	if !ok {
		return types.ExportResult{Success: false, Error: "that edition is no longer part of this book"}
	}
	if len(request.Items) == 0 {
		return types.ExportResult{Success: false, Error: "choose at least one format to export"}
	}

	title := strings.TrimSpace(b.Metadata.Title)
	if title == "" {
		title = "Untitled"
	}
	root := export.SafeName(strings.TrimSpace(title + " — " + strings.TrimSpace(edition.Label)))

	target, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export this edition",
		DefaultFilename: root + ".zip",
		Filters: []runtime.FileFilter{
			{DisplayName: "Zip archives (*.zip)", Pattern: "*.zip"},
		},
	})
	if err != nil || target == "" {
		return types.ExportResult{Success: false, Error: "cancelled"}
	}
	if !strings.HasSuffix(strings.ToLower(target), ".zip") {
		target += ".zip"
	}

	members, err := a.bundleMembers(b, request, root, title)
	if err != nil {
		return types.ExportResult{Success: false, Error: err.Error()}
	}
	return export.Bundle(target, members)
}

// bundleMembers renders every chosen format and gathers the artwork that goes
// with it. It builds the whole list before anything is written.
func (a *App) bundleMembers(
	b types.BookData, request types.BundleRequest, root, title string,
) ([]export.BundleMember, error) {
	var members []export.BundleMember
	taken := map[string]bool{}

	for _, item := range request.Items {
		edition, format, ok := b.Editions.FindFormat(strings.TrimSpace(item.FormatID))
		if !ok {
			return nil, fmt.Errorf("one of the chosen formats is no longer part of this book")
		}
		word := strings.TrimSpace(format.Format)
		if word == "" {
			word = outputWord(item.Output)
		}
		folder := path.Join(root, export.UniqueFolder(taken, word))

		data, name, err := a.renderBundleItem(b, item)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", word, err)
		}
		members = append(members, export.BundleMember{
			Path: path.Join(folder, export.SafeName(title)+name),
			Data: data,
		})

		if !request.IncludeArtwork {
			continue
		}
		art, err := a.bundleArtwork(edition, format, item.Output, folder)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", word, err)
		}
		members = append(members, art...)
	}
	return members, nil
}

// renderBundleItem is the exporter for one format, and the suffix its file
// takes. A hardcover interior is the print exporter at the hardcover's own
// measurements; an audiobook script is the reading-copy exporter, because
// Draftline writes no audio and what a narrator reads from is a PDF.
func (a *App) renderBundleItem(b types.BookData, item types.BundleItem) ([]byte, string, error) {
	source, err := a.exportSource(b, item.Shared)
	if err != nil {
		return nil, "", err
	}
	switch item.Output {
	case "epub":
		data, err := export.EPUBBytes(source, item.EPUB, a.exportCover(b, item.EPUB.ExportOptions))
		return data, ".epub", err
	case "docx":
		data, err := export.DOCXBytes(source, item.Shared)
		return data, ".docx", err
	case "pdf":
		data, err := export.PDFBytes(source, item.PDF, a.exportCover(b, item.PDF.ExportOptions))
		return data, " — reading copy.pdf", err
	case "audio":
		// A narration script, not the reading copy renamed: its own page, its
		// own aids, its own options.
		data, err := export.AudioScriptBytes(source, item.Audio, a.exportCover(b, item.Audio.ExportOptions))
		return data, " — audiobook script.pdf", err
	case "hc":
		data, err := export.PrintPDFBytes(source, item.Print)
		return data, " — hardcover interior.pdf", err
	default:
		data, err := export.PrintPDFBytes(source, item.Print)
		return data, " — print interior.pdf", err
	}
}

// bundleArtwork is what travels beside one interior.
//
// A printed object gets the wrap and only the wrap: the full wraparound at the
// spine width of that binding, which is the author's own file and is handed
// back exactly as it arrived. A paperback's wrap and a hardcover's wrap are
// different files at different spine widths, and neither is the cover.
//
// An ebook gets the storefront cover. An audiobook script gets the same cover,
// because that is the image on the page it opens with.
func (a *App) bundleArtwork(
	edition types.Edition, format types.EditionFormat, output, folder string,
) ([]export.BundleMember, error) {
	if output == "docx" || output == "pdf" {
		return nil, nil
	}
	if output == "print-pdf" || output == "hc" {
		wrap := format.Wrap
		if wrap == nil {
			return nil, nil
		}
		name := export.SafeName(strings.TrimSpace(wrap.FileName))
		if wrap.Stored && strings.TrimSpace(wrap.Member) != "" {
			data, err := a.coverBytes(edition.ID, filepath.Base(wrap.Member))
			if err != nil {
				return nil, err
			}
			if len(data) == 0 {
				return nil, fmt.Errorf("the wrap artwork kept in this project could not be read")
			}
			return []export.BundleMember{{Path: path.Join(folder, name), Data: data}}, nil
		}
		// Not kept in the project: the record says where it lives, and the
		// bundle copies it through from there without loading it whole.
		source := strings.TrimSpace(wrap.SourcePath)
		if source == "" {
			return nil, nil
		}
		return []export.BundleMember{{Path: path.Join(folder, name), Source: source}}, nil
	}

	cover := edition.Cover
	if cover == nil || strings.TrimSpace(cover.File) == "" {
		return nil, nil
	}
	data, err := a.coverBytes(edition.ID, cover.File)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return []export.BundleMember{{Path: path.Join(folder, export.SafeName(cover.File)), Data: data}}, nil
}

// outputWord names a folder for a format whose record carries no word of its
// own, which a record being filled in can legitimately be.
func outputWord(output string) string {
	switch output {
	case "epub":
		return "eBook"
	case "docx":
		return "Manuscript"
	case "pdf":
		return "Reading copy"
	case "hc":
		return "Hardcover"
	case "audio":
		return "Audiobook"
	default:
		return "Paperback"
	}
}
