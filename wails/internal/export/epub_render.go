package export

import (
	"fmt"
	"net/url"
	"strings"

	"draftline/internal/types"
)

func renderEPUBSection(doc Document, section DocumentSection, options types.EPUBOptions) string {
	var body strings.Builder
	body.WriteString("    <section epub:type=\"")
	body.WriteString(epubSectionType(section.Role))
	body.WriteString("\">\n")
	if strings.TrimSpace(section.Title) != "" {
		fmt.Fprintf(&body, "      <h1>%s</h1>\n", EscapeXML(section.Title))
	}
	if strings.TrimSpace(section.Subtitle) != "" {
		fmt.Fprintf(&body, "      <p class=\"subtitle\">%s</p>\n", EscapeXML(section.Subtitle))
	}

	listOpen := false
	listOrdered := false
	closeList := func() {
		if !listOpen {
			return
		}
		if listOrdered {
			body.WriteString("      </ol>\n")
		} else {
			body.WriteString("      </ul>\n")
		}
		listOpen = false
	}
	for _, block := range section.Blocks {
		if block.Kind == BlockListItem {
			if !listOpen || listOrdered != block.Ordered {
				closeList()
				listOpen = true
				listOrdered = block.Ordered
				if listOrdered {
					body.WriteString("      <ol>\n")
				} else {
					body.WriteString("      <ul>\n")
				}
			}
			fmt.Fprintf(&body, "        <li%s>%s</li>\n", epubAlignmentClass(block.Alignment), renderEPUBRuns(block.Runs))
			continue
		}
		closeList()
		renderEPUBBlock(&body, block, options)
	}
	closeList()
	body.WriteString("    </section>\n")

	title := displaySectionTitle(section.Title)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="%s" lang="%s">
<head><meta charset="UTF-8"/><title>%s</title><link rel="stylesheet" type="text/css" href="../styles/book.css"/></head>
<body class="%s %s">
%s  </body>
</html>`, EscapeXML(doc.Language), EscapeXML(doc.Language), EscapeXML(title), epubRoleClass(section.Role), EscapeXML(options.ChapterStyle), body.String())
}

func renderEPUBBlock(out *strings.Builder, block DocumentBlock, options types.EPUBOptions) {
	content := renderEPUBRuns(block.Runs)
	align := epubAlignmentClass(block.Alignment)
	switch block.Kind {
	case BlockHeading:
		level := block.Level
		if level < 2 || level > 6 {
			level = 2
		}
		fmt.Fprintf(out, "      <h%d%s>%s</h%d>\n", level, align, content, level)
	case BlockBlockquote:
		fmt.Fprintf(out, "      <blockquote%s><p>%s</p></blockquote>\n", align, content)
	case BlockCode:
		fmt.Fprintf(out, "      <pre%s><code>%s</code></pre>\n", align, content)
	case BlockSceneBreak:
		switch options.SceneBreakStyle {
		case "rule":
			out.WriteString("      <hr class=\"scene-break scene-rule\"/>\n")
		case "space":
			out.WriteString("      <div class=\"scene-break scene-space\" aria-hidden=\"true\"></div>\n")
		default:
			out.WriteString("      <div class=\"scene-break scene-asterism\" aria-hidden=\"true\">⁂</div>\n")
		}
	default:
		fmt.Fprintf(out, "      <p%s>%s</p>\n", align, content)
	}
}

func renderEPUBRuns(runs []DocumentRun) string {
	var out strings.Builder
	for _, run := range runs {
		if run.LineBreak {
			out.WriteString("<br/>")
			continue
		}
		text := EscapeXML(run.Text)
		if text == "" {
			continue
		}
		if run.Code {
			text = "<code>" + text + "</code>"
		}
		if run.Superscript {
			text = "<sup>" + text + "</sup>"
		}
		if run.Subscript {
			text = "<sub>" + text + "</sub>"
		}
		if run.Strike {
			text = "<s>" + text + "</s>"
		}
		if run.Underline {
			text = "<span class=\"underline\">" + text + "</span>"
		}
		if run.Italic {
			text = "<em>" + text + "</em>"
		}
		if run.Bold {
			text = "<strong>" + text + "</strong>"
		}
		if href := safeEPUBHref(run.Href); href != "" {
			text = "<a href=\"" + EscapeXML(href) + "\">" + text + "</a>"
		}
		out.WriteString(text)
	}
	return out.String()
}

func safeEPUBHref(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "", "http", "https", "mailto":
		return raw
	default:
		return ""
	}
}

func epubAlignmentClass(alignment string) string {
	switch normalizedAlignment(alignment, "") {
	case "left", "center", "right", "justify":
		return " class=\"align-" + normalizedAlignment(alignment, "") + "\""
	default:
		return ""
	}
}

func epubSectionType(role SectionRole) string {
	switch role {
	case SectionCopyright:
		return "copyright-page"
	case SectionFront:
		return "frontmatter"
	case SectionBack:
		return "backmatter"
	default:
		return "chapter"
	}
}

func epubRoleClass(role SectionRole) string {
	switch role {
	case SectionCopyright:
		return "copyright"
	case SectionFront:
		return "front-matter"
	case SectionBack:
		return "back-matter"
	default:
		return "body-matter"
	}
}

func renderEPUBCSS(options types.EPUBOptions) string {
	var css strings.Builder
	css.WriteString(`@charset "UTF-8";
html { -webkit-hyphens: auto; hyphens: auto; }
body { margin: 5%; line-height: 1.5; widows: 2; orphans: 2; }
section { max-width: 42em; margin: 0 auto; }
h1 { text-align: center; break-after: avoid; page-break-after: avoid; }
.classic h1 { margin: 22vh 0 2.5em; }
.minimal h1 { margin: 1.5em 0 1.75em; text-align: left; }
.subtitle { text-align: center; font-style: italic; text-indent: 0 !important; }
blockquote { margin: 1em 1.5em; }
pre { white-space: pre-wrap; font-family: monospace; }
.underline { text-decoration: underline; }
.align-left { text-align: left !important; }
.align-center { text-align: center !important; }
.align-right { text-align: right !important; }
.align-justify { text-align: justify !important; }
.scene-break { break-inside: avoid; page-break-inside: avoid; text-align: center; }
.scene-asterism { margin: 1.5em 0; }
.scene-rule { border: 0; border-top: 1px solid currentColor; margin: 1.75em auto; width: 18%; }
.scene-space { height: 2em; }
.scene-break + p, h1 + p, h2 + p, h3 + p, blockquote + p { text-indent: 0; }
nav ol { padding-left: 1.5em; }
nav li { margin: 0.35em 0; }
`)
	if options.ParagraphStyle == "spaced" {
		css.WriteString("p { margin: 0 0 0.8em; text-indent: 0; }\n")
	} else {
		css.WriteString("p { margin: 0; text-indent: 1.35em; }\n")
	}
	switch options.TextAlign {
	case "left", "justify":
		fmt.Fprintf(&css, "p { text-align: %s; }\n", options.TextAlign)
	}
	if family, ok := embeddedPDFFonts[options.FontFamily]; ok {
		name := EscapeXML(family.DisplayName)
		fmt.Fprintf(&css, `@font-face { font-family: '%s'; src: url('../fonts/%s-regular.ttf') format('truetype'); font-weight: 400; font-style: normal; }
@font-face { font-family: '%s'; src: url('../fonts/%s-bold.ttf') format('truetype'); font-weight: 700; font-style: normal; }
@font-face { font-family: '%s'; src: url('../fonts/%s-italic.ttf') format('truetype'); font-weight: 400; font-style: italic; }
@font-face { font-family: '%s'; src: url('../fonts/%s-bold-italic.ttf') format('truetype'); font-weight: 700; font-style: italic; }
body { font-family: '%s', serif; }
`, name, options.FontFamily, name, options.FontFamily, name, options.FontFamily, name, options.FontFamily, name)
	}
	return css.String()
}
