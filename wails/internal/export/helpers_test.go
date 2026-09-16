package export

import (
	"regexp"
	"strings"
	"testing"
)

func TestHtmlToPlainParagraphsDecodesEntities(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "named ampersand",
			in:   "<p>Smith &amp; Sons</p>",
			want: "Smith & Sons",
		},
		{
			name: "nbsp becomes regular space",
			in:   "<p>Chapter&nbsp;One</p>",
			want: "Chapter One",
		},
		{
			name: "em dash",
			in:   "<p>wait&mdash;stop</p>",
			want: "wait—stop",
		},
		{
			name: "numeric right single quote",
			in:   "<p>it&#8217;s here</p>",
			want: "it’s here",
		},
		{
			name: "mixed entities across paragraphs",
			in:   "<p>Tom &amp; Jerry&nbsp;&mdash;&nbsp;that&#8217;s all</p><p>&lt;tag&gt; stays text</p>",
			want: "Tom & Jerry — that’s all\n\n<tag> stays text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HtmlToPlainParagraphs(tt.in)
			if got != tt.want {
				t.Errorf("HtmlToPlainParagraphs(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHtmlToPlainParagraphsLeavesNoRawNbsp(t *testing.T) {
	got := HtmlToPlainParagraphs("<p>a&nbsp;b</p>")
	if strings.ContainsRune(got, ' ') {
		t.Errorf("output still contains U+00A0: %q", got)
	}
}

// TestDocxPathNoDoubleEscaping exercises the DOCX pipeline order
// (HtmlToPlainParagraphs, then EscapeXML) and verifies entities are not
// double-escaped: &amp; in source HTML decodes to & and is then re-escaped
// exactly once for XML output.
func TestDocxPathNoDoubleEscaping(t *testing.T) {
	in := "<p>Smith &amp; Sons &mdash; it&#8217;s &lt;fine&gt;</p>"
	plain := HtmlToPlainParagraphs(in)
	got := EscapeXML(plain)
	want := "Smith &amp; Sons — it’s &lt;fine&gt;"
	if got != want {
		t.Errorf("EscapeXML(HtmlToPlainParagraphs(%q)) = %q, want %q", in, got, want)
	}
	if strings.Contains(got, "&amp;amp;") {
		t.Errorf("double-escaped ampersand in %q", got)
	}
}

func TestEscapePDFStringEncodesPublishingPunctuationAsWinAnsi(t *testing.T) {
	got := []byte(EscapePDFString("\u201cIt\u2019s\u2014fine\u2026\u201d (caf\u00e9) \\"))
	want := []byte{0x93, 'I', 't', 0x92, 's', 0x97, 'f', 'i', 'n', 'e', 0x85, 0x94, ' ', '\\', '(', 'c', 'a', 'f', 0xe9, '\\', ')', ' ', '\\', '\\'}
	if string(got) != string(want) {
		t.Fatalf("EscapePDFString bytes = % x, want % x", got, want)
	}
	if strings.Contains(string(got), "\u2019") || strings.Contains(string(got), "\u2014") {
		t.Fatal("PDF literal still contains raw UTF-8 publishing punctuation")
	}
}

func TestEscapePDFStringReplacesUnsupportedGlyphs(t *testing.T) {
	if got := EscapePDFString("Latin \u03a9 CJK \u6f22"); got != "Latin ? CJK ?" {
		t.Fatalf("unsupported glyph fallback = %q", got)
	}
}

// An EPUB with no ISBN identifies itself by a UUID, and that identifier is how
// a reading device decides whether the file it has been handed is the book it
// already holds. The old one was cut out of a single nanosecond timestamp, so
// two exports made in the same moment could carry the same identifier.
func TestGenerateUUIDIsRandomAndConformant(t *testing.T) {
	shape := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		id := GenerateUUID()
		if !shape.MatchString(id) {
			t.Fatalf("not an RFC 4122 version 4 UUID: %q", id)
		}
		if seen[id] {
			t.Fatalf("two exports produced the same identifier: %q", id)
		}
		seen[id] = true
	}
}
