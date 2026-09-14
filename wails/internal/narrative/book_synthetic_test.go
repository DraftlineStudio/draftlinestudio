package narrative

import (
	"reflect"
	"testing"

	"draftline/internal/types"
)

// Locks the evaluation passages to the real manuscript, not to fabricated
// summaries. This does not execute or claim the unimplemented braid gates.

func TestHTMLInlineFormattingDoesNotChangeEvidence(t *testing.T) {
	b := types.BookData{Body: []types.ChapterItem{{ID: "ch1", Content: `<p>Mara sat with Elias in the interrogation room.</p><p>Elias sat alone in the interrogation room.</p>`}}}
	d1, err := BodyDocument(b, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	b.Body[0].Content = `<p><em>Mara</em> sat with Elias in the interrogation room.</p><p>Elias sat <strong>alone</strong> in the interrogation room.</p>`
	d2, err := BodyDocument(b, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d1, d2) {
		t.Fatal("inline formatting changed canonical evidence")
	}
	_, w1 := inspect(t, d1)
	_, w2 := inspect(t, d2)
	if !reflect.DeepEqual(w1, w2) {
		t.Fatal("formatting changed finding")
	}
}

func TestHTMLSceneAndDocumentBoundaries(t *testing.T) {
	for _, separator := range []string{`<hr>`, `<p>* * *</p>`} {
		b := types.BookData{Body: []types.ChapterItem{{ID: "ch1", Content: `<p>Mara sat with Elias in the interrogation room.</p>` + separator + `<p>Elias sat alone in the interrogation room.</p>`}}}
		d, err := BodyDocument(b, 0, 1)
		if err != nil {
			t.Fatal(err)
		}
		_, w := inspect(t, d)
		if len(w) != 0 {
			t.Fatal("presence leaked across scene break")
		}
	}
	b := types.BookData{Body: []types.ChapterItem{{ID: "ch1", Content: `<blockquote><p>Mara sat with Elias in the interrogation room.</p></blockquote><pre>Mara entered the room.</pre><p>Elias sat alone in the interrogation room.</p>`}}}
	d, err := BodyDocument(b, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	e, w := inspect(t, d)
	if len(w) != 0 {
		t.Fatal("embedded text became presence")
	}
	for _, o := range e.Observations {
		if o.Subject == "mara" {
			t.Fatal("embedded text emitted occupancy")
		}
	}
}
