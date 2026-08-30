package indexing

import (
	"strings"

	"github.com/jdkato/prose/v3"
)

// linguisticSpan is prose/v3's local classification of a byte range. The
// model is embedded in the Draftline binary: no manuscript text leaves the
// process and no generative model is involved.
type linguisticSpan struct {
	start int
	end   int
	label string
}

type linguisticEvidence struct {
	spans []linguisticSpan
}

func analyzeLinguisticEvidence(text string) linguisticEvidence {
	// Named-entity extraction requires tokenization and POS tagging, but not
	// sentence segmentation. Models are immutable and shared process-wide by
	// prose, so repeated chapter analysis does not reload them.
	doc, err := prose.NewDocument(text, prose.WithSegmentation(false))
	if err != nil {
		// Deterministic extraction remains a safe fallback if NLP analysis ever
		// rejects malformed input.
		return linguisticEvidence{}
	}

	evidence := linguisticEvidence{spans: make([]linguisticSpan, 0, len(doc.Entities()))}
	for _, entity := range doc.Entities() {
		evidence.spans = append(evidence.spans, linguisticSpan{
			start: entity.Start,
			end:   entity.End(),
			label: strings.ToUpper(entity.Label),
		})
	}
	return evidence
}

// classification returns the strongest prose label overlapping a candidate.
// PERSON wins over a partially overlapping non-person span because titles and
// common-word trimming can make Draftline's candidate narrower than prose's.
func (e linguisticEvidence) classification(start, end int) (person, nonPerson bool) {
	for _, span := range e.spans {
		if span.end <= start || span.start >= end {
			continue
		}
		if span.label == "PERSON" {
			person = true
		} else {
			nonPerson = true
		}
	}
	if person {
		return true, false
	}
	return false, nonPerson
}
