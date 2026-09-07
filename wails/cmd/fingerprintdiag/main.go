// Command fingerprintdiag prints Draftline's three textual manuscript-memory
// quality reports for one or more .draftline archives. It reads archives and
// their persisted evidence only; it never edits or saves a manuscript.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"draftline/internal/book"
	"draftline/internal/fingerprint"
	"draftline/internal/types"
)

func main() {
	summary := flag.Bool("summary", false, "print report headings and counts without details")
	reportKind := flag.String("report", "all", "report to print: corpus, developments, inspections, or all")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/fingerprintdiag [-summary] [-report corpus|developments|inspections|all] manuscript.draftline [...]")
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
		reports := fingerprint.TextDiagnostics(&manuscript)
		fmt.Printf("=== %s ===\n", path)
		selected, ok := selectReports(reports, *reportKind)
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown report %q; use corpus, developments, inspections, or all\n", *reportKind)
			failed = true
			continue
		}
		for _, report := range selected {
			if *summary {
				fmt.Println(reportSummary(report))
			} else {
				fmt.Print(report)
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

func selectReports(reports types.FingerprintTextDiagnostics, kind string) ([]string, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "corpus":
		return []string{reports.Corpus}, true
	case "developments", "development":
		return []string{reports.Developments}, true
	case "inspections", "inspection":
		return []string{reports.Inspections}, true
	case "all", "":
		return []string{reports.Corpus, reports.Developments, reports.Inspections}, true
	default:
		return nil, false
	}
}

func reportSummary(report string) string {
	trimmed := strings.TrimSpace(report)
	lines := strings.Split(report, "\n")
	result := make([]string, 0, 12)
	for _, line := range lines {
		if strings.Contains(line, "DIAGNOSTIC") || strings.HasPrefix(line, "Engine:") || strings.HasPrefix(line, "Evidence atoms:") || strings.HasPrefix(line, "Assertions:") || strings.HasPrefix(line, "Fingerprints:") || strings.HasPrefix(line, "Relations:") || strings.HasPrefix(line, "Same-event identities:") || strings.HasPrefix(line, "State histories:") || strings.HasPrefix(line, "Developments:") || strings.HasPrefix(line, "Inspections:") {
			result = append(result, line)
		}
	}
	if len(result) == 0 {
		return trimmed
	}
	return strings.Join(result, "\n")
}
