// narrativediag runs the isolated source/presence prototype. It does not save
// books, rebuild entity analysis, write reports, or touch production fingerprints.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"draftline/internal/book"
	"draftline/internal/narrative"
)

func main() {
	from := flag.Int("from", 0, "first body chapter, zero-based")
	to := flag.Int("to", 0, "exclusive last body chapter (0 means all)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: narrativediag [-from 0 -to 4] manuscript.draftline")
		os.Exit(2)
	}
	b, err := book.Open(flag.Arg(0))
	check(err)
	end := *to
	if end == 0 {
		end = len(b.Body)
	}
	d, err := narrative.BodyDocument(b, *from, end)
	check(err)
	people := narrative.AcceptedPeople(b)
	extraction, err := narrative.ExtractPresence(d, people)
	check(err)
	inspections, err := narrative.InspectPresence(d, extraction)
	check(err)
	report := struct {
		Stage       string                 `json:"stage"`
		Document    narrative.Document     `json:"document"`
		RosterSize  int                    `json:"roster_size"`
		Extraction  narrative.Extraction   `json:"extraction"`
		Inspections []narrative.Inspection `json:"inspections"`
	}{"experimental-presence-only-not-braid-acceptance", d, len(people), extraction, inspections}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	check(encoder.Encode(report))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
