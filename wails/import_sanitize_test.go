package main

import (
	"strings"
	"testing"
)

// sanitize is a test helper: parse a document and join the blocks.
func sanitize(t *testing.T, doc string) string {
	t.Helper()
	_, blocks, _, err := parseSpineDoc([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	return joinBlocks(blocks)
}

func TestSanitizeSpineDoc(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains []string
		absent   []string
	}{
		{
			name:     "loose div text becomes paragraphs",
			in:       `<body><div>First stretch.<div>Nested stretch.</div>Tail stretch.</div></body>`,
			contains: []string{"<p>First stretch.</p>", "<p>Nested stretch.</p>", "<p>Tail stretch.</p>"},
			absent:   []string{"<div"},
		},
		{
			name:     "sections unwrap without merging paragraphs",
			in:       `<body><section><p>One.</p></section><section><p>Two.</p></section></body>`,
			contains: []string{"<p>One.</p>\n<p>Two.</p>"},
			absent:   []string{"<section"},
		},
		{
			name:     "legacy inline tags map to editor marks",
			in:       `<body><p><b>bold</b> and <i>italic</i> and <strike>gone</strike></p></body>`,
			contains: []string{"<strong>bold</strong>", "<em>italic</em>", "<s>gone</s>"},
			absent:   []string{"<b>", "<i>", "<strike>"},
		},
		{
			name:     "spans anchors and classes unwrap to text",
			in:       `<body><p class="x" id="y"><span class="dropcap">O</span>nce upon <a href="notes.xhtml#fn1" epub:type="noteref">a time</a>.</p></body>`,
			contains: []string{"<p>Once upon a time.</p>"},
			absent:   []string{"<span", "<a ", "class=", "id=", "epub:"},
		},
		{
			name:     "deep headings clamp to h3",
			in:       `<body><h4>Scene IV</h4><h6>Deep</h6><p>Prose.</p></body>`,
			contains: []string{"<h3>Scene IV</h3>", "<h3>Deep</h3>"},
			absent:   []string{"<h4", "<h6"},
		},
		{
			name:     "images svg and scripts drop entirely",
			in:       `<body><p>Before.</p><img src="../images/x.jpg" alt="art"/><svg><text>vector words</text></svg><script>alert(1)</script><p>After.</p></body>`,
			contains: []string{"<p>Before.</p>", "<p>After.</p>"},
			absent:   []string{"img", "svg", "vector words", "alert"},
		},
		{
			name:     "missing closing body cannot leak head or style",
			in:       `<html><head><title>Book</title><style>p { color: red }</style></head><body><p>Real prose.`,
			contains: []string{"<p>Real prose.</p>"},
			absent:   []string{"color: red", "<style", "Book"},
		},
		{
			name: "entities decode and re-escape safely",
			in:   `<body><p>Em &mdash; dash &#8212; twice &lt;tag&gt;</p></body>`,
			// &lt;tag&gt; is literal text "<tag>", so it must re-escape —
			// never resurface as markup.
			contains: []string{"Em — dash — twice &lt;tag&gt;"},
			absent:   []string{"<tag>"},
		},
		{
			name:     "pre preserves internal whitespace",
			in:       "<body><pre>line one\n  indented two\n\nline three</pre></body>",
			contains: []string{"<pre><code>line one\n  indented two\n\nline three</code></pre>"},
		},
		{
			name:     "br verse survives inside paragraphs",
			in:       `<body><p>roses are red<br/>violets are blue</p></body>`,
			contains: []string{"roses are red<br>violets are blue"},
		},
		{
			name:   "empty and whitespace-only paragraphs drop",
			in:     `<body><p></p><p>   </p><p><br/><br/></p><p>Kept.</p></body>`,
			absent: []string{"<p></p>", "<p> </p>"},
			contains: []string{
				"<p>Kept.</p>",
			},
		},
		{
			name:     "scene break hr survives",
			in:       `<body><p>One.</p><hr class="scene"/><p>Two.</p></body>`,
			contains: []string{"<p>One.</p>\n<hr>\n<p>Two.</p>"},
		},
		{
			name:     "text-align survives on paragraphs and headings",
			in:       `<body><h1 style="text-align:center; margin: 0">Title</h1><p style="TEXT-ALIGN: Right">End.</p></body>`,
			contains: []string{`<h1 style="text-align: center">Title</h1>`, `<p style="text-align: right">End.</p>`},
			absent:   []string{"margin"},
		},
		{
			name:     "table cells become paragraphs",
			in:       `<body><table><tr><td>Cell one.</td><td>Cell two.</td></tr></table></body>`,
			contains: []string{"<p>Cell one.</p>", "<p>Cell two.</p>"},
			absent:   []string{"<table", "<td"},
		},
		{
			name:     "lists survive with nested marks",
			in:       `<body><ul><li>First <b>item</b></li><li>Second</li></ul></body>`,
			contains: []string{"<ul><li>First <strong>item</strong></li><li>Second</li></ul>"},
		},
		{
			name:     "blockquote keeps inner paragraphs",
			in:       `<body><blockquote><p>Quoted line.</p></blockquote></body>`,
			contains: []string{"<blockquote><p>Quoted line.</p></blockquote>"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitize(t, tc.in)
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("missing %q in:\n%s", want, got)
				}
			}
			for _, banned := range tc.absent {
				if strings.Contains(got, banned) {
					t.Fatalf("unexpected %q in:\n%s", banned, got)
				}
			}
		})
	}
}

func TestSanitizeCountsDroppedImages(t *testing.T) {
	_, _, dropped, err := parseSpineDoc([]byte(`<body><p>Text.</p><img src="a.jpg"/><div><img src="b.jpg"/></div><svg></svg></body>`))
	if err != nil {
		t.Fatal(err)
	}
	if dropped != 3 {
		t.Fatalf("expected 3 dropped images, got %d", dropped)
	}
}

func TestDecodeToUTF8(t *testing.T) {
	utf16le := func(s string) []byte {
		out := []byte{0xFF, 0xFE}
		for _, r := range s {
			out = append(out, byte(r), byte(r>>8)) // BMP-only test input
		}
		return out
	}
	utf16be := func(s string) []byte {
		out := []byte{0xFE, 0xFF}
		for _, r := range s {
			out = append(out, byte(r>>8), byte(r))
		}
		return out
	}

	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"utf8 passthrough", []byte("plain café"), "plain café"},
		{"utf8 bom stripped", append([]byte{0xEF, 0xBB, 0xBF}, []byte("bom café")...), "bom café"},
		{"utf16le bom", utf16le("wide café"), "wide café"},
		{"utf16be bom", utf16be("wide café"), "wide café"},
		{
			"declared latin-1",
			append([]byte(`<?xml version="1.0" encoding="iso-8859-1"?><p>caf`), 0xE9, '<', '/', 'p', '>'),
			`<?xml version="1.0" encoding="iso-8859-1"?><p>café</p>`,
		},
		{
			"invalid utf8 falls back to windows-1252",
			[]byte{'c', 'a', 'f', 0xE9},
			"café",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := string(decodeToUTF8(tc.in)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
