// plotdiag reads prose and writes a standalone evidence report to stdout only.
package main

import (
	"draftline/internal/book"
	"draftline/internal/narrative"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	from := flag.Int("from", 0, "first body entry, zero-based")
	to := flag.Int("to", 0, "exclusive final body entry; zero means all")
	format := flag.String("format", "json", "json or html")
	flag.Parse()
	check := func(err error) {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: plotdiag [-from N -to N] [-format json|html] manuscript.draftline")
		os.Exit(2)
	}
	if *format != "json" && *format != "html" {
		check(fmt.Errorf("unknown output format"))
	}
	b, err := book.Open(flag.Arg(0))
	check(err)
	end := *to
	if end == 0 {
		end = len(b.Body)
	}
	d, err := narrative.BodyDocument(b, *from, end)
	check(err)
	p, err := narrative.AnalyzePlot(d)
	check(err)
	if *format == "html" {
		page, err := narrative.RenderPlotHTML(d, p)
		check(err)
		fmt.Print(page)
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	check(enc.Encode(struct {
		Document narrative.Document     `json:"document"`
		Analysis narrative.PlotAnalysis `json:"analysis"`
	}{d, p}))
}
