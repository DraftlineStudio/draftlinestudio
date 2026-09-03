package indexing

import (
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
	return analyzeBookLinguisticEvidenceWithOptions(chapters, defaultAnalysisPoolOptions())
}

type linguisticBatch struct {
	start int
	end   int
	bytes int
}

func analyzeBookLinguisticEvidenceWithOptions(chapters []string, pool AnalysisPoolOptions) []linguisticEvidence {
	result := make([]linguisticEvidence, len(chapters))
	if len(chapters) == 0 {
		return result
	}
	batches := linguisticBatches(chapters, pool)
	pool = normalizeAnalysisPoolOptions(pool, len(batches))
	jobs := make(chan linguisticBatch)
	memory := newAnalysisMemoryGate(pool.MaxInFlightBytes)
	var workers sync.WaitGroup
	for range pool.Workers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for batch := range jobs {
				weight := memory.acquire(batch.bytes)
				analyzeChapterBatch(chapters, result, batch.start, batch.end)
				memory.release(weight)
			}
		}()
	}
	for _, batch := range batches {
		jobs <- batch
	}
	close(jobs)
	workers.Wait()
	return result
}

func linguisticBatches(chapters []string, pool AnalysisPoolOptions) []linguisticBatch {
	pool = normalizeAnalysisPoolOptions(pool, len(chapters))
	targetCount := pool.Workers
	defaultBatchChapters := (len(chapters) + targetCount - 1) / targetCount
	result := []linguisticBatch{}
	for start := 0; start < len(chapters); {
		end, bytes := start, 0
		for end < len(chapters) {
			nextBytes := len(chapters[end])
			if end > start && ((pool.MaxBatchBytes > 0 && bytes+nextBytes > pool.MaxBatchBytes) || end-start >= defaultBatchChapters) {
				break
			}
			bytes += nextBytes
			end++
			if pool.MaxBatchBytes > 0 && bytes >= pool.MaxBatchBytes {
				break
			}
		}
		result = append(result, linguisticBatch{start: start, end: end, bytes: bytes})
		start = end
	}
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
