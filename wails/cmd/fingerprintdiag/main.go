// Command fingerprintdiag writes Draftline's three textual manuscript-memory
// diagnostics for a .draftline archive:
//
//	fingerprints-v5.txt            — the typed frame corpus, ledgers, identities
//	narrative-developments-v5.txt  — the synthesized story account
//	inspections-v5.txt             — continuity findings with scope assessment
//
// It reads archives and their persisted evidence only; it never edits or
// saves a manuscript. Files are written UTF-8.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"draftline/internal/book"
	"draftline/internal/fingerprint"
)

func main() {
	outDir := flag.String("out", ".", "directory to write the three diagnostic files into")
	stdout := flag.Bool("stdout", false, "print the reports to stdout instead of writing files")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/fingerprintdiag [-out dir | -stdout] manuscript.draftline")
		os.Exit(2)
	}
	path := flag.Arg(0)
	manuscript, err := book.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}
	reports := fingerprint.TextDiagnostics(&manuscript)
	files := []struct {
		name    string
		content string
	}{
		{"fingerprints-v5.txt", reports.Frames},
		{"narrative-developments-v5.txt", reports.Developments},
		{"inspections-v5.txt", reports.Inspections},
	}
	if *stdout {
		for _, file := range files {
			fmt.Printf("=== %s ===\n%s\n", file.name, file.content)
		}
		return
	}
	for _, file := range files {
		target := filepath.Join(*outDir, file.name)
		if err := os.WriteFile(target, []byte(file.content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", target, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d bytes)\n", target, len(file.content))
	}
}
