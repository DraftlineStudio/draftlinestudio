package narrative

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jdkato/prose/v3"
)

// PlotClaim separates discourse acts from assertions about physical reality.
// A quoted report is a disclosure, not narrator-confirmed knowledge or fact.
type PlotClaim struct {
	Observation  Observation `json:"observation"`
	Target       string      `json:"target,omitempty"`
	TargetSource *Anchor     `json:"target_source,omitempty"`
	LinkStatus   string      `json:"link_status,omitempty"`
}

type PlotAnalysis struct {
	Graph           Graph            `json:"graph"`
	Claims          []PlotClaim      `json:"claims"`
	Attributes      []AttributeClaim `json:"attributes"`
	UncoveredBlocks []string         `json:"uncovered_blocks"`
	Limitations     []string         `json:"limitations"`
}

var situationReport = regexp.MustCompile(`(?i)\bwe have (?:a|an) (.+?) at ([^.!?]+)`)
var commitment = regexp.MustCompile(`(?i)^(?:I['’]ll take it[.!]?|Responding(?: on foot)?[.!]?)$`)
var ongoingTask = regexp.MustCompile(`(?i)\b(.+?) is still (?:handling|working on) the (.+?) (?:call|task|assignment) at ([^.!?]+)`)
var searchRequest = regexp.MustCompile(`(?i)^Requesting (?:a |an )?((?:(?:thorough|full|complete|systematic) )?search (?:of|for) [^.!?]+)[.!]?$`)
var completionReport = regexp.MustCompile(`(?i)\bI (?:cleared|completed|finished) that (call|task|assignment)\b`)
var negativeFinding = regexp.MustCompile(`(?i)(?:\b(?:they|we) found nothing\b|\bthere was no (?:sign|trace) of [^.!?]+|\bno body (?:located|found)\b)`)
var recordDispute = regexp.MustCompile(`(?i)\b[^.!?]*has no record of (?:that|any) call[^.!?]*`)
var priorSearch = regexp.MustCompile(`(?i)\bI['’]ve been (?:looking|searching)[^.!?]* for you\b`)
var disclosedRecord = regexp.MustCompile(`(?i)\b(?:the screen|the monitor) (?:showed|displayed) [^.!?]+`)
var quotedText = regexp.MustCompile(`“([^“”]*)”|"([^"\n]*)"`)

type plotSpan struct {
	start, end int
	mode       string
}

// Keeps quote boundaries, not speakers. Missing attribution remains explicit;
// this prototype does not replace the existing dialogue-attribution engine.
func plotSpans(text string) []plotSpan {
	if !balancedMovementQuotes(text) {
		return nil
	}
	var result []plotSpan
	start := 0
	for _, match := range quotedText.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > start {
			result = append(result, plotSpan{start, match[0], "current_narration"})
		}
		a, z := match[2], match[3]
		if a < 0 {
			a, z = match[4], match[5]
		}
		result = append(result, plotSpan{a, z, "unattributed_report"})
		start = match[1]
	}
	if start < len(text) {
		result = append(result, plotSpan{start, len(text), "current_narration"})
	}
	return result
}

func taskAddress(s string) string {
	s = strings.ToLower(strings.Trim(strings.TrimSpace(s), ".,"))
	r := strings.NewReplacer("west ", "w ", "east ", "e ", "north ", "n ", "south ", "s ")
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}

// AnalyzePlot is independent of movement/presence acceptance gates. It is a
// bounded automatic discourse/activity slice, not full plot comprehension.
func AnalyzePlot(d Document) (PlotAnalysis, error) {
	if err := d.Validate(); err != nil {
		return PlotAnalysis{}, err
	}
	p := PlotAnalysis{Claims: []PlotClaim{}, Attributes: []AttributeClaim{}, UncoveredBlocks: []string{}, Limitations: []string{
		"Partial automatic activity projection, not a passed braided-timeline acceptance test.",
		"Quoted statements retain unresolved speakers; no reader disclosure is assigned as character knowledge.",
		"Dashed links are candidate task references, not established causal relations.",
		"Search outcomes, reunion, and disputed records are not joined by shared words or cast.",
		"An open strand means its outcome was not linked in this excerpt, not an unresolved author plot hole.",
	}}
	var observations []Observation
	var nodes []Development
	var edges []Relation
	type activity struct{ id, node, address string }
	var activities []activity
	var previousQuoteReport *PlotClaim
	previousQuoteScene := ""
	for bi, b := range d.Blocks {
		if b.Mode != "current_narration" {
			p.UncoveredBlocks = append(p.UncoveredBlocks, b.ID)
			continue
		}
		// Extraction of one claim never asserts full understanding of a block.
		p.UncoveredBlocks = append(p.UncoveredBlocks, b.ID)
		for _, span := range plotSpans(b.Text) {
			seg, err := prose.NewDocument(b.Text[span.start:span.end], prose.WithExtraction(false), prose.WithTagging(false), prose.WithTokenization(false))
			if err != nil {
				return PlotAnalysis{}, err
			}
			var thisQuoteReport *PlotClaim
			for _, sentence := range seg.Sentences() {
				text := strings.TrimSpace(sentence.Text)
				if unsafeContext.MatchString(text) || strings.HasSuffix(text, "?") {
					continue
				}
				base := span.start + sentence.Start + len(sentence.Text) - len(strings.TrimLeft(sentence.Text, " \t\r\n"))
				kind, target := "", ""
				var targetSource *Anchor
				if span.mode == "unattributed_report" {
					if m := situationReport.FindStringSubmatchIndex(text); m != nil {
						kind = "situation_report"
						target = text[m[4]:m[5]]
						a := d.Anchor(bi, base+m[4], base+m[5])
						targetSource = &a
					} else if commitment.MatchString(text) {
						kind = "response_commitment"
					} else if m := ongoingTask.FindStringSubmatchIndex(text); m != nil {
						kind = "ongoing_activity_report"
						target = text[m[6]:m[7]]
						a := d.Anchor(bi, base+m[6], base+m[7])
						targetSource = &a
					} else if m := searchRequest.FindStringSubmatchIndex(text); m != nil {
						kind = "search_request"
						target = text[m[2]:m[3]]
						a := d.Anchor(bi, base+m[2], base+m[3])
						targetSource = &a
					} else if completionReport.MatchString(text) {
						kind = "reported_task_completion"
					} else if recordDispute.MatchString(text) {
						kind = "record_dispute"
					} else if priorSearch.MatchString(text) {
						kind = "reported_prior_search"
					}
				}
				if kind == "" && negativeFinding.MatchString(text) {
					kind = "negative_finding"
				}
				if kind == "" && span.mode == "current_narration" && disclosedRecord.MatchString(text) {
					kind = "record_disclosure"
				}
				if kind == "" {
					continue
				}
				if kind == "response_commitment" && (previousQuoteReport == nil || previousQuoteScene != b.Scene) {
					continue
				}
				source := d.Anchor(bi, base, base+len(text))
				o := Observation{ID: id(d.Revision, b.ID, fmt.Sprint(base), kind), Kind: kind, Scene: b.Scene, Mode: span.mode, Source: source, Rule: "plot-discourse-v1"}
				c := PlotClaim{Observation: o, Target: target, TargetSource: targetSource}
				n := Development{ID: id("plot-node", o.ID), Label: kind + ": " + text, EstablishedBy: o.ID}
				if kind == "situation_report" {
					copy := c
					thisQuoteReport = &copy
				}
				if kind == "response_commitment" || kind == "search_request" {
					a := activity{id: id("activity", o.ID), node: n.ID}
					n.Changes = []ActivityChange{{ActivityID: a.id, Operation: "initiate", Witness: o.ID}}
					if kind == "response_commitment" && previousQuoteReport != nil && previousQuoteScene == b.Scene {
						// Adjacency of conversational turns is candidate anaphora,
						// not certainty of task identity or the speaker's identity.
						a.address = taskAddress(previousQuoteReport.Target)
						c.Target = previousQuoteReport.Target
						c.TargetSource = previousQuoteReport.TargetSource
						c.LinkStatus = "candidate_previous_turn_reference"
						n.SupportingObservations = []string{previousQuoteReport.Observation.ID}
						edges = append(edges, Relation{ID: id("edge", previousQuoteReport.Observation.ID, o.ID), From: id("plot-node", previousQuoteReport.Observation.ID), To: n.ID, Kind: "branches", Witnesses: []string{previousQuoteReport.Observation.ID, o.ID}, Rule: c.LinkStatus})
					}
					activities = append(activities, a)
				}
				if kind == "ongoing_activity_report" {
					matches := []int{}
					for i, a := range activities {
						if a.address != "" && a.address == taskAddress(target) {
							matches = append(matches, i)
						}
					}
					if len(matches) == 1 {
						a := &activities[matches[0]]
						c.LinkStatus = "candidate_unique_address_task_reference"
						n.Changes = []ActivityChange{{ActivityID: a.id, Operation: "continue", Witness: o.ID}}
						edges = append(edges, Relation{ID: id("edge", a.node, n.ID), From: a.node, To: n.ID, Kind: "reports", Witnesses: []string{o.ID}, Rule: c.LinkStatus})
						a.node = n.ID
					}
				}
				p.Claims = append(p.Claims, c)
				observations = append(observations, o)
				nodes = append(nodes, n)
			}
			if span.mode == "unattributed_report" {
				previousQuoteReport = thisQuoteReport
				previousQuoteScene = b.Scene
			}
		}
	}
	var err error
	p.Graph, err = BuildGraph(d, observations, nodes, edges)
	if err != nil {
		return PlotAnalysis{}, err
	}
	p.Attributes = extractAttributes(d)
	return p, nil
}
