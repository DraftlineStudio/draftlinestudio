// Package ai provides AI rewriting utilities and helpers.
package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// HTMLBlock is one top-level block of chapter HTML: a prose paragraph or a
// structural block (scene-break <hr>, <blockquote>, <pre>, heading, list).
type HTMLBlock struct {
	// Tag is the lower-case element name; "p" for prose and bare text.
	Tag string
	// HTML is the verbatim block markup, nested children included.
	HTML string
}

var (
	blockTokenRe = regexp.MustCompile(`(?is)<!--.*?-->|<(/?)([a-z][a-z0-9-]*)(?:\s[^>]*)?(/?)>`)
	voidTags     = map[string]bool{"hr": true, "br": true, "img": true, "input": true, "wbr": true, "source": true, "col": true, "embed": true, "area": true, "base": true, "link": true, "meta": true, "param": true, "track": true}
	inlineTags   = map[string]bool{"em": true, "strong": true, "b": true, "i": true, "u": true, "s": true, "strike": true, "span": true, "a": true, "code": true, "sub": true, "sup": true, "mark": true, "small": true, "del": true, "ins": true, "br": true}
)

// ExtractHTMLBlocks splits HTML into its top-level blocks. Nested markup (a
// <p> inside a <blockquote>, <li> inside <ul>, <code> inside <pre>) stays
// inside its parent block so structure survives a diff round trip. Bare
// top-level text and inline markup become a paragraph block. Mirrors
// splitTopLevelBlocks in the frontend diff utility.
func ExtractHTMLBlocks(html string) []HTMLBlock {
	var blocks []HTMLBlock
	depth := 0
	blockStart := -1
	blockTag := ""
	cursor := 0
	pushBareText := func(to int) {
		if raw := strings.TrimSpace(html[cursor:to]); raw != "" {
			blocks = append(blocks, HTMLBlock{Tag: "p", HTML: raw})
		}
	}
	for _, loc := range blockTokenRe.FindAllStringSubmatchIndex(html, -1) {
		if strings.HasPrefix(html[loc[0]:loc[1]], "<!--") {
			continue
		}
		isClose := loc[2] >= 0 && loc[3]-loc[2] == 1
		name := strings.ToLower(html[loc[4]:loc[5]])
		selfClosing := (loc[6] >= 0 && loc[7]-loc[6] == 1) || voidTags[name]
		end := loc[1]
		switch {
		case depth == 0:
			if isClose || inlineTags[name] {
				continue // stray close tag or inline prose at the top level
			}
			pushBareText(loc[0])
			if selfClosing {
				blocks = append(blocks, HTMLBlock{Tag: name, HTML: html[loc[0]:end]})
				cursor = end
				continue
			}
			blockStart = loc[0]
			blockTag = name
			depth = 1
		case isClose:
			depth--
			if depth == 0 {
				blocks = append(blocks, HTMLBlock{Tag: blockTag, HTML: html[blockStart:end]})
				cursor = end
			}
		case !selfClosing:
			depth++
		}
	}
	if depth > 0 {
		// Unterminated block (malformed model output): keep everything from its start.
		blocks = append(blocks, HTMLBlock{Tag: blockTag, HTML: strings.TrimSpace(html[blockStart:])})
		cursor = len(html)
	}
	pushBareText(len(html))
	return blocks
}

// BuildDiffUserMsg prefixes each top-level block with §N§ so the model can
// return only changed blocks by index instead of the full chapter.
func BuildDiffUserMsg(html string) string {
	blocks := ExtractHTMLBlocks(html)
	var sb strings.Builder
	for i, b := range blocks {
		sb.WriteString(fmt.Sprintf("§%d§%s\n", i+1, b.HTML))
	}
	return sb.String()
}

// acceptBlockChange decides whether a model's replacement for one block may
// be spliced in. Prose paragraphs take whatever came back. Structural blocks
// only accept a replacement that keeps their kind: a scene break is never
// replaced, a block quote that came back as bare paragraphs is re-wrapped,
// and anything else that changed kind is dropped so the original stays.
func acceptBlockChange(original HTMLBlock, changed string) (string, bool) {
	if original.Tag == "p" {
		return changed, true
	}
	if original.Tag == "hr" {
		return "", false
	}
	replacement := ExtractHTMLBlocks(changed)
	if len(replacement) == 0 {
		return "", false
	}
	sameKind := true
	allParagraphs := true
	for _, b := range replacement {
		if b.Tag != original.Tag {
			sameKind = false
		}
		if b.Tag != "p" {
			allParagraphs = false
		}
	}
	if sameKind {
		return changed, true
	}
	if original.Tag == "blockquote" && allParagraphs {
		return "<blockquote>" + changed + "</blockquote>", true
	}
	return "", false
}

// ApplyDiffResponse parses §N§<block> AI output and splices the changed
// blocks back into the original HTML, returning a complete HTML string.
// Structural blocks the model omitted, or returned as a different kind of
// block, are kept verbatim from the original.
func ApplyDiffResponse(originalHTML, aiResponse string) string {
	if strings.Contains(aiResponse, "§NONE§") || strings.TrimSpace(aiResponse) == "" {
		return originalHTML
	}
	blocks := ExtractHTMLBlocks(originalHTML)
	if len(blocks) == 0 {
		return originalHTML
	}

	// Split on § — the response is "§N§content§N§content…" so splitting yields
	// ["", "1", "content", "5", "content", …]. Process odd/even index pairs.
	parts := strings.Split(aiResponse, "§")
	changes := make(map[int]string)
	for i := 1; i+1 < len(parts); i += 2 {
		idx, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || idx < 1 || idx > len(blocks) {
			continue
		}
		content := strings.TrimSpace(parts[i+1])
		if content == "" {
			continue
		}
		if accepted, ok := acceptBlockChange(blocks[idx-1], content); ok {
			changes[idx-1] = accepted
		}
	}

	var sb strings.Builder
	for i, b := range blocks {
		if changed, ok := changes[i]; ok {
			sb.WriteString(changed)
		} else {
			sb.WriteString(b.HTML)
		}
		if i < len(blocks)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
