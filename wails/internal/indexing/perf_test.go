package indexing

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"draftline/internal/types"
)

// buildLargeBook generates a ~1MB manuscript with 200 distinct characters.
func buildLargeBook(chapters, parasPerChapter int) types.BookData {
	first := []string{"Marcus", "Clara", "Daniel", "Kira", "Elena", "Tobias", "Ingrid", "Rafael", "Sofia", "Viktor", "Amara", "Dmitri", "Lucia", "Henrik", "Yara", "Callum", "Nadia", "Emeric", "Talia", "Bram"}
	last := []string{"Webb", "Chen", "Hanlon", "Morrow", "Ruiz", "Abernathy", "Kessler", "Vance", "Ito", "Duarte"}
	var names []string
	for _, f := range first {
		for _, l := range last {
			names = append(names, f+" "+l)
		}
	}
	var book types.BookData
	for c := 0; c < chapters; c++ {
		var sb strings.Builder
		for p := 0; p < parasPerChapter; p++ {
			n1 := names[(c*31+p*7)%len(names)]
			n2 := names[(c*17+p*13+3)%len(names)]
			short1 := strings.Fields(n1)[0]
			sb.WriteString(fmt.Sprintf(`<p>%s crossed the room slowly and deliberately. "You should not have come here tonight," %s said. %s's blue eyes narrowed as the rain hammered the windows outside. They argued about the case for a long while before %s finally gave up and left.</p>`, n1, n2, short1, short1))
		}
		book.Body = append(book.Body, types.ChapterItem{Title: fmt.Sprintf("Ch%d", c), Type: "chapter", Content: sb.String()})
	}
	return book
}

// Indexing a full-length manuscript must complete in seconds, never minutes.
// Guards against reintroducing per-character full-text scans or quadratic
// prefix work (the causes of a 15-minute index on real books).
func TestIndexBook_LargeBookPerformance(t *testing.T) {
	book := buildLargeBook(80, 45)

	start := time.Now()
	result := IndexBook(&book)
	elapsed := time.Since(start)

	if !result.Success {
		t.Fatalf("IndexBook failed: %s", result.Error)
	}

	// Distinct people sharing a first name must not collapse into one
	// entity via bare first-name reference mentions.
	if got := len(book.Analysis.EntityResolution.Entities); got != 200 {
		t.Errorf("expected 200 distinct characters, got %d", got)
	}

	// 10x headroom over the measured ~1.3s to stay CI-safe.
	if elapsed > 15*time.Second {
		t.Errorf("indexing a ~1MB book took %s — performance regression (must stay in seconds)", elapsed)
	}
	t.Logf("indexed ~1MB / 200 characters / %d mentions in %s", len(book.Analysis.EntityResolution.Mentions), elapsed)
}
