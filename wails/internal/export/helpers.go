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

// EscapePDFString converts Unicode prose to the WinAnsi byte encoding used by
// the built-in PDF fonts, then escapes PDF literal-string delimiters. Writing
// raw UTF-8 bytes while declaring WinAnsi makes curly quotes and em dashes
// render as mojibake. The replacement renderer will embed full Unicode fonts;
// this compatibility path preserves the publishing punctuation WinAnsi has.
func EscapePDFString(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		encoded, ok := winAnsiByte(r)
		if !ok {
			encoded = '?'
		}
		if encoded == '\\' || encoded == '(' || encoded == ')' {
			out.WriteByte('\\')
		}
		out.WriteByte(encoded)
	}
	return out.String()
}

func winAnsiByte(r rune) (byte, bool) {
	if r >= 0x20 && r <= 0x7e {
		return byte(r), true
	}
	if r >= 0xa0 && r <= 0xff {
		return byte(r), true
	}
	if r == '\t' || r == '\n' || r == '\r' || r == '\u202f' || r == '\u2009' {
		return ' ', true
	}

	switch r {
	case '\u20ac':
		return 0x80, true
	case '\u201a':
		return 0x82, true
	case '\u0192':
		return 0x83, true
	case '\u201e':
		return 0x84, true
	case '\u2026':
		return 0x85, true
	case '\u2020':
		return 0x86, true
	case '\u2021':
		return 0x87, true
	case '\u02c6':
		return 0x88, true
	case '\u2030':
		return 0x89, true
	case '\u0160':
		return 0x8a, true
	case '\u2039':
		return 0x8b, true
	case '\u0152':
		return 0x8c, true
	case '\u017d':
		return 0x8e, true
	case '\u2018':
		return 0x91, true
	case '\u2019':
		return 0x92, true
	case '\u201c':
		return 0x93, true
	case '\u201d':
		return 0x94, true
	case '\u2022':
		return 0x95, true
	case '\u2013':
		return 0x96, true
	case '\u2014':
		return 0x97, true
	case '\u02dc':
		return 0x98, true
	case '\u2122':
		return 0x99, true
	case '\u0161':
		return 0x9a, true
	case '\u203a':
		return 0x9b, true
	case '\u0153':
		return 0x9c, true
	case '\u017e':
		return 0x9e, true
	case '\u0178':
		return 0x9f, true
	case '\u2010', '\u2011', '\u2212':
		return '-', true
	case '\u200b', '\u2060':
		return ' ', true
	default:
		return 0, false
	}
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
