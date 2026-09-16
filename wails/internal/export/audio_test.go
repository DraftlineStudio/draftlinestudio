package export

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func scriptBook() types.BookData {
	return types.BookData{
		Metadata: types.Metadata{Title: "Wide Water", Author: "A. Marsh"},
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter",
				Content: "<p>She counted the lamps along the pier.</p><hr /><p>By dawn the tide had turned.</p>"},
			{Title: "Chapter Two", Type: "chapter",
				Content: "<p>" + strings.Repeat("The boat came back light. ", 40) + "</p>"},
		},
	}
}

// A narration script is its own document, not the reading copy renamed. If
// these two ever render identically, the script has stopped being one.
func TestNarrationScriptIsNotTheReadingCopy(t *testing.T) {
	book := scriptBook()
	script, err := AudioScriptBytes(book, types.AudioOptions{
		PageSize: "letter", FontFamily: "lato", FontSize: 14, LineHeight: 1.8,
		ParagraphSpacing: "one line between", SlatePage: true, NumberParagraphs: true,
		PauseBreaks: true, PronunciationColumn: true, ChapterWordCount: true,
	}, nil)
	if err != nil {
		t.Fatalf("rendering the script: %v", err)
	}
	reading, err := PDFBytes(book, types.PDFOptions{
		PageSize: "letter", FontFamily: "merriweather", FontSize: 12, LineHeight: 1.5,
	}, nil)
	if err != nil {
		t.Fatalf("rendering the reading copy: %v", err)
	}
	if len(script) == 0 {
		t.Fatal("the script is empty")
	}
	if string(script) == string(reading) {
		t.Fatal("the narration script rendered identically to the reading copy")
	}
}

// Every narration aid has to actually change the page. A toggle that renders
// the same file whether it is on or off is a dead control, which is what the
// audiobook settings were before this.
func TestEveryNarrationAidChangesThePage(t *testing.T) {
	base := types.AudioOptions{
		PageSize: "letter", FontFamily: "lato", FontSize: 14, LineHeight: 1.8,
		ParagraphSpacing: "one line between",
	}
	plain, err := AudioScriptBytes(scriptBook(), base, nil)
	if err != nil {
		t.Fatalf("rendering the plain script: %v", err)
	}

	aids := map[string]func(*types.AudioOptions){
		"slate page":           func(o *types.AudioOptions) { o.SlatePage = true },
		"numbered paragraphs":  func(o *types.AudioOptions) { o.NumberParagraphs = true },
		"pause breaks":         func(o *types.AudioOptions) { o.PauseBreaks = true },
		"pronunciation column": func(o *types.AudioOptions) { o.PronunciationColumn = true },
		"chapter word count":   func(o *types.AudioOptions) { o.SlatePage, o.ChapterWordCount = true, true },
		"paragraph spacing":    func(o *types.AudioOptions) { o.ParagraphSpacing = "two lines between" },
		"type size":            func(o *types.AudioOptions) { o.FontSize = 16 },
		"line spacing":         func(o *types.AudioOptions) { o.LineHeight = 2.0 },
		"page size":            func(o *types.AudioOptions) { o.PageSize = "a4" },
		"typeface":             func(o *types.AudioOptions) { o.FontFamily = "merriweather" },
	}
	for name, apply := range aids {
		options := base
		apply(&options)
		changed, err := AudioScriptBytes(scriptBook(), options, nil)
		if err != nil {
			t.Fatalf("rendering with %s: %v", name, err)
		}
		if string(changed) == string(plain) {
			t.Errorf("%s changed nothing in the script", name)
		}
	}
}

// The spacing words the screen offers are the ones the script understands.
func TestParagraphSpacingWords(t *testing.T) {
	for words, want := range map[string]float64{
		"half line between": 0.5,
		"one line between":  1,
		"two lines between": 2,
		"":                  1,
	} {
		if got := audioParagraphSpacing(words); got != want {
			t.Errorf("audioParagraphSpacing(%q) = %v, wanted %v", words, got, want)
		}
	}
}
