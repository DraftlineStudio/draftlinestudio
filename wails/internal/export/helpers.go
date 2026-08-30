// Package export provides multi-format book export functionality.
package export

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

// HtmlToPlainParagraphs converts HTML content to plain text paragraphs.
func HtmlToPlainParagraphs(content string) string {
	// Replace paragraph and heading tags with newlines
	text := regexp.MustCompile(`</?(p|h[1-6]|div|br)[^>]*>`).ReplaceAllString(content, "\n")
	// Remove remaining HTML tags
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
	// Decode HTML entities (&amp; &nbsp; &mdash; &#8217; ...) into their characters
	// so exporters receive plain text, not markup escapes.
	text = html.UnescapeString(text)
	// &nbsp; decodes to U+00A0; normalize to a regular space for export output.
	text = strings.ReplaceAll(text, " ", " ")
	// Clean up multiple newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// EscapePDFString escapes special characters for PDF string literals.
func EscapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// EscapeXML escapes special characters for XML content.
func EscapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// GenerateUUID creates a simple UUID-like string for EPUB identifiers.
func GenerateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano()&0xFFFFFFFF,
		time.Now().UnixNano()>>32&0xFFFF,
		0x4000|(time.Now().UnixNano()>>48&0x0FFF),
		0x8000|(time.Now().UnixNano()>>60&0x3FFF),
		time.Now().UnixNano()&0xFFFFFFFFFFFF)
}
