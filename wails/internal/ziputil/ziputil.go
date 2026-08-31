// Package ziputil enforces resource limits when reading untrusted ZIP
// archives (.draftline files, EPUB/DOCX imports, downloaded tooling).
package ziputil

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrEntryNotFound is returned by ReadNamed when no entry matches the requested
// name. Callers use errors.Is to distinguish an absent entry from a read
// failure (e.g. an oversized or corrupt entry that does exist).
var ErrEntryNotFound = errors.New("entry not found in archive")

const (
	// MaxEntrySize caps a single decompressed entry. Chapters are HTML/JSON;
	// 50 MB is far beyond any real manuscript component.
	MaxEntrySize = 50 << 20
	// MaxEntries caps the archive's file count.
	MaxEntries = 10_000
	// MaxTotalSize caps the declared total uncompressed size of the archive.
	MaxTotalSize = 500 << 20
	// MaxCompressionRatio guards against decompression bombs. Only enforced
	// for entries whose declared uncompressed size exceeds ratioFloor, since
	// tiny files legitimately compress extremely well.
	MaxCompressionRatio = 200
	ratioFloor          = 1 << 20
)

// CheckArchive validates header-level limits across an archive's file list.
// Declared sizes can lie, so ReadEntry independently enforces MaxEntrySize on
// the bytes actually read.
func CheckArchive(files []*zip.File) error {
	if len(files) > MaxEntries {
		return fmt.Errorf("archive has %d entries (limit %d)", len(files), MaxEntries)
	}
	var total uint64
	for _, f := range files {
		if f.UncompressedSize64 > MaxEntrySize {
			return fmt.Errorf("entry %q declares %d bytes (limit %d)", f.Name, f.UncompressedSize64, MaxEntrySize)
		}
		total += f.UncompressedSize64
		if total > MaxTotalSize {
			return fmt.Errorf("archive declares more than %d bytes uncompressed", MaxTotalSize)
		}
		if f.UncompressedSize64 > ratioFloor && f.CompressedSize64 > 0 &&
			f.UncompressedSize64/f.CompressedSize64 > MaxCompressionRatio {
			return fmt.Errorf("entry %q exceeds compression ratio limit", f.Name)
		}
	}
	return nil
}

// ReadEntry reads one entry, failing if the actual decompressed content
// exceeds MaxEntrySize regardless of what the header declares.
func ReadEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(io.LimitReader(rc, MaxEntrySize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxEntrySize {
		return nil, fmt.Errorf("entry %q exceeds %d bytes", f.Name, MaxEntrySize)
	}
	return data, nil
}

// ReadNamed finds an entry by name (optionally case-insensitively) and reads
// it with ReadEntry limits. Returns an error if the entry is absent.
func ReadNamed(files []*zip.File, name string, caseInsensitive bool) ([]byte, error) {
	for _, f := range files {
		if f.Name == name || (caseInsensitive && strings.EqualFold(f.Name, name)) {
			return ReadEntry(f)
		}
	}
	return nil, fmt.Errorf("%q: %w", name, ErrEntryNotFound)
}
