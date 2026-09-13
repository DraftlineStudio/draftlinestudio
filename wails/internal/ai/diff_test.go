package ai

import (
	"strings"
	"testing"
)

const structuredChapter = `<p>One.</p><p></p><hr><p></p><p>Two.</p><blockquote><p>Quoted <em>line</em>.</p></blockquote><pre><code>E:/USB
README.txt</code></pre><h3>Later</h3><ul><li>a</li><li>b</li></ul><p>Three.</p>`

// Scene breaks, block quotes, code blocks, headings, and lists are blocks of
// their own kind; nested paragraphs stay inside their wrapper.
func TestExtractHTMLBlocksKeepsStructure(t *testing.T) {
	blocks := ExtractHTMLBlocks(structuredChapter)
	var tags []string
	for _, b := range blocks {
		tags = append(tags, b.Tag)
	}
	if got, want := strings.Join(tags, " "), "p p hr p p blockquote pre h3 ul p"; got != want {
		t.Fatalf("block tags = %q, want %q", got, want)
	}
	if blocks[5].HTML != "<blockquote><p>Quoted <em>line</em>.</p></blockquote>" {
		t.Fatalf("blockquote block lost its wrapper: %q", blocks[5].HTML)
	}
	if !strings.HasPrefix(blocks[6].HTML, "<pre><code>") || !strings.HasSuffix(blocks[6].HTML, "</code></pre>") {
		t.Fatalf("pre block = %q", blocks[6].HTML)
	}
	if strings.Join(blocksHTML(blocks), "") != structuredChapter {
		t.Fatalf("blocks do not round-trip the source verbatim")
	}
}

func TestExtractHTMLBlocksWrapsBareText(t *testing.T) {
	blocks := ExtractHTMLBlocks("Plain <em>inline</em> prose")
	if len(blocks) != 1 || blocks[0].Tag != "p" || blocks[0].HTML != "Plain <em>inline</em> prose" {
		t.Fatalf("bare text blocks = %+v", blocks)
	}
	if got := ExtractHTMLBlocks("   "); len(got) != 0 {
		t.Fatalf("whitespace produced blocks: %+v", got)
	}
}

// The numbered user message must carry every block, so the model sees
// structure in place and paragraph indices stay aligned with the original.
func TestBuildDiffUserMsgNumbersEveryBlock(t *testing.T) {
	msg := BuildDiffUserMsg("<p>One.</p><hr><p>Two.</p>")
	if !strings.Contains(msg, "§1§<p>One.</p>") || !strings.Contains(msg, "§2§<hr>") || !strings.Contains(msg, "§3§<p>Two.</p>") {
		t.Fatalf("unexpected diff message:\n%s", msg)
	}
}

// A model that omits the scene break, or rewrites it as a paragraph, must
// not remove it from the reconstructed chapter.
func TestApplyDiffResponsePreservesSceneBreak(t *testing.T) {
	original := "<p>One.</p><hr><p>Two.</p>"
	got := ApplyDiffResponse(original, "§1§<p>One!</p>\n§2§<p></p>\n§3§<p>Two!</p>")
	if got != "<p>One!</p>\n<hr>\n<p>Two!</p>" {
		t.Fatalf("scene break lost: %q", got)
	}
}

func TestApplyDiffResponseKeepsStructuralKinds(t *testing.T) {
	original := "<blockquote><p>Quote.</p></blockquote><pre><code>x = 1</code></pre><h3>Head</h3><p>Body.</p>"

	// Same-kind replacements are accepted; a block quote returned as bare
	// paragraphs is re-wrapped; a code block or heading returned as a
	// paragraph is refused so the original stays.
	got := ApplyDiffResponse(original, "§1§<p>Quote, fixed.</p>\n§2§<p>x = 1</p>\n§3§<p>Head</p>\n§4§<p>Body, fixed.</p>")
	want := "<blockquote><p>Quote, fixed.</p></blockquote>\n<pre><code>x = 1</code></pre>\n<h3>Head</h3>\n<p>Body, fixed.</p>"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}

	got = ApplyDiffResponse(original, "§2§<pre><code>x = 2</code></pre>\n§3§<h3>Heading</h3>")
	if !strings.Contains(got, "<pre><code>x = 2</code></pre>") || !strings.Contains(got, "<h3>Heading</h3>") {
		t.Fatalf("same-kind structural edits refused: %q", got)
	}
}

func TestApplyDiffResponseNoneKeepsOriginal(t *testing.T) {
	if got := ApplyDiffResponse(structuredChapter, "§NONE§"); got != structuredChapter {
		t.Fatalf("§NONE§ altered the chapter")
	}
}

func blocksHTML(blocks []HTMLBlock) []string {
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, b.HTML)
	}
	return out
}
