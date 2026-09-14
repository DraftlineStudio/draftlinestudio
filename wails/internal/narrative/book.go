package narrative

import (
	"fmt"
	"strings"

	"draftline/internal/types"
	"golang.org/x/net/html"
)

// BodyDocument reads raw manuscript blocks, not potentially stale EvidenceData.
// Chapter boundaries conservatively end presence scope in this first slice;
// continuity across them will require an explicit supported context relation.
func BodyDocument(book types.BookData, from, to int) (Document, error) {
	if from < 0 || to <= from || to > len(book.Body) {
		return Document{}, fmt.Errorf("invalid body range [%d,%d)", from, to)
	}
	blocks := []Block{}
	for chapterIndex := from; chapterIndex < to; chapterIndex++ {
		chapter := book.Body[chapterIndex]
		root, err := html.Parse(strings.NewReader(chapter.Content))
		if err != nil {
			return Document{}, err
		}
		ordinal, scene := 0, 0
		chapterKey := chapter.ID
		if chapterKey == "" {
			chapterKey = fmt.Sprintf("body/%03d", chapterIndex)
		}
		var plain func(*html.Node) string
		plain = func(n *html.Node) string {
			if n.Type == html.TextNode {
				return n.Data
			}
			if n.Type == html.ElementNode && n.Data == "br" {
				return "\n"
			}
			var b strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				b.WriteString(plain(c))
			}
			return b.String()
		}
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if n.Data == "hr" {
					scene++
					return
				}
				switch n.Data {
				case "p", "pre", "h1", "h2", "h3", "h4", "h5", "h6":
					text := strings.TrimSpace(plain(n))
					mode := "current_narration"
					if n.Data == "pre" {
						mode = "embedded_document"
					} else if n.Data != "p" {
						mode = "metadata"
					}
					for parent := n.Parent; parent != nil; parent = parent.Parent {
						if parent.Type == html.ElementNode && parent.Data == "blockquote" {
							mode = "quoted"
						}
					}
					if text == "***" || text == "* * *" || text == "⁂" {
						scene++
						ordinal++
						return
					}
					if text != "" {
						blocks = append(blocks, Block{ID: fmt.Sprintf("%s/block/%d", chapterKey, ordinal), Scene: fmt.Sprintf("%s/scene/%d", chapterKey, scene), Text: text, Mode: mode})
					}
					ordinal++
					return
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
		walk(root)
	}
	return NewDocument(blocks), nil
}

// AcceptedPeople uses the existing roster unchanged. Missing/ambiguous entities
// remain an explicit coverage limitation; the prototype does not invent people.
func AcceptedPeople(book types.BookData) []Person {
	result := []Person{}
	for _, c := range book.StoryBible.Characters {
		if c.ID == "" || c.Name == "" || (c.EntityKind != "" && c.EntityKind != "person") || c.DetectionStatus == "rejected" {
			continue
		}
		if c.IsAutoDetected && c.DetectionStatus != "accepted" {
			continue
		}
		result = append(result, Person{ID: c.ID, Names: append([]string{c.Name}, c.Aliases...)})
	}
	return result
}
