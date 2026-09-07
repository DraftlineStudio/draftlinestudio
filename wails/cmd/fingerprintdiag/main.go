// Command fingerprintdiag prints Draftline's textual narrative-fingerprint
// quality report for one or more .draftline archives. It reads archives and
// their persisted evidence only; it never edits or saves a manuscript.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"draftline/internal/book"
	"draftline/internal/fingerprint"
)

func main() {
	summary := flag.Bool("summary", false, "print report counts without fingerprint details")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/fingerprintdiag [-summary] manuscript.draftline [...]")
		os.Exit(2)
	}
	failed := false
	for _, path := range flag.Args() {
		manuscript, err := book.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failed = true
			continue
		}
		report := fingerprint.DiagnosticReport(&manuscript)
		fmt.Printf("=== %s ===\n", path)
		if *summary {
			fmt.Println(reportSummary(report))
		} else {
			fmt.Print(report)
		}
	}
	if failed {
		os.Exit(1)
	}
}

func reportSummary(report string) string {
	trimmed := strings.TrimSpace(report)
	lines := strings.Split(report, "\n")
	result := make([]string, 0, 6)
	for _, line := range lines {
		if strings.HasPrefix(line, "Engine:") || strings.HasPrefix(line, "Evidence atoms:") || strings.HasPrefix(line, "Assertions:") || strings.HasPrefix(line, "Promoted fingerprints:") || strings.HasPrefix(line, "Retained as evidence only:") {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return trimmed
	}
	return strings.Join(result, "\n")
}
