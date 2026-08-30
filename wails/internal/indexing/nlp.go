package indexing

import (
	"runtime"
	"strings"
	"sync"

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

// analyzeBookLinguisticEvidence classifies bounded chapter batches concurrently.
// prose models are immutable and shared process-wide, so batching amortizes
// document setup while the worker limit avoids multiplying model memory.
func analyzeBookLinguisticEvidence(chapters []string) []linguisticEvidence {
	result := make([]linguisticEvidence, len(chapters))
	if len(chapters) == 0 {
		return result
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount > 8 {
		workerCount = 8
	}
	if workerCount > len(chapters) {
		workerCount = len(chapters)
	}

	var workers sync.WaitGroup
	batchSize := (len(chapters) + workerCount - 1) / workerCount
	for start := 0; start < len(chapters); start += batchSize {
		end := start + batchSize
		if end > len(chapters) {
			end = len(chapters)
		}
		workers.Add(1)
		go func(start, end int) {
			defer workers.Done()
			analyzeChapterBatch(chapters, result, start, end)
		}(start, end)
	}
	workers.Wait()
	return result
}

func analyzeChapterBatch(chapters []string, result []linguisticEvidence, start, end int) {
	starts := make([]int, end-start)
	var combined strings.Builder
	for index := start; index < end; index++ {
		if index > start {
			combined.WriteString("\n\n")
		}
		starts[index-start] = combined.Len()
		combined.WriteString(chapters[index])
	}

	batchEvidence := analyzeLinguisticEvidence(combined.String())
	localChapter := 0
	for _, span := range batchEvidence.spans {
		for localChapter+1 < len(starts) && span.start >= starts[localChapter]+len(chapters[start+localChapter]) {
			localChapter++
		}
		chapterStart := starts[localChapter]
		chapterEnd := chapterStart + len(chapters[start+localChapter])
		if span.start < chapterStart || span.end > chapterEnd {
			continue
		}
		result[start+localChapter].spans = append(result[start+localChapter].spans, linguisticSpan{
			start: span.start - chapterStart,
			end:   span.end - chapterStart,
			label: span.label,
		})
	}
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
