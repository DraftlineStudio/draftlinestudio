// Package ai provides AI rewriting utilities and helpers.
package ai

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ExtractHTMLParagraphs extracts <p>...</p> elements from HTML content.
func ExtractHTMLParagraphs(html string) []string {
	re := regexp.MustCompile(`(?is)<p[^>]*>[\s\S]*?</p>`)
	paras := re.FindAllString(html, -1)
	if len(paras) == 0 && strings.TrimSpace(html) != "" {
		return []string{"<p>" + strings.TrimSpace(html) + "</p>"}
	}
	return paras
}

// BuildDiffUserMsg prefixes each paragraph with §N§ so the model can return
// only changed paragraphs by index instead of the full chapter.
func BuildDiffUserMsg(html string) string {
	paras := ExtractHTMLParagraphs(html)
	var sb strings.Builder
	for i, p := range paras {
		sb.WriteString(fmt.Sprintf("§%d§%s\n", i+1, p))
	}
	return sb.String()
}

// ApplyDiffResponse parses §N§<p>...</p> AI output and splices the changed
// paragraphs back into the original HTML, returning a complete HTML string.
func ApplyDiffResponse(originalHTML, aiResponse string) string {
	if strings.Contains(aiResponse, "§NONE§") || strings.TrimSpace(aiResponse) == "" {
		return originalHTML
	}
	paras := ExtractHTMLParagraphs(originalHTML)
	if len(paras) == 0 {
		return originalHTML
	}

	// Split on § — the response is "§N§content§N§content…" so splitting yields
	// ["", "1", "content", "5", "content", …]. Process odd/even index pairs.
	parts := strings.Split(aiResponse, "§")
	changes := make(map[int]string)
	for i := 1; i+1 < len(parts); i += 2 {
		idx, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || idx < 1 || idx > len(paras) {
			continue
		}
		content := strings.TrimSpace(parts[i+1])
		if content != "" {
			changes[idx-1] = content
		}
	}

	var sb strings.Builder
	for i, p := range paras {
		if changed, ok := changes[i]; ok {
			sb.WriteString(changed)
		} else {
			sb.WriteString(p)
		}
		if i < len(paras)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
