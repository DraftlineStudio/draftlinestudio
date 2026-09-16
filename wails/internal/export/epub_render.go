package export

import (
	"fmt"
	"net/url"
	"strings"

	"draftline/internal/types"
)

// epubDocumentShell is the XHTML a content document is wrapped in.
//
// This is where the two EPUBs genuinely part company. An EPUB 3 content
// document is XHTML5: the HTML5 doctype, <meta charset>, and the epub:
// namespace that carries epub:type. An OPS 2.0.1 content document is XHTML
// 1.1, where every one of those is a validation error — which is the whole
// point of choosing EPUB 2.0.1 in the first place, since the reading systems
// that need it are the ones that will not open anything else.
func epubDocumentShell(profile epubProfile, language, title, bodyAttrs, head, body string) string {
	lang := EscapeXML(language)
	if profile.Three {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="%s" lang="%s">
<head><meta charset="UTF-8"/><title>%s</title>%s</head>
<body%s>
%s</body>
</html>`, lang, lang, EscapeXML(title), head, bodyAttrs, body)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="%s">
<head><meta http-equiv="Content-Type" content="application/xhtml+xml; charset=utf-8"/><title>%s</title>%s</head>
<body%s>
%s</body>
</html>`, lang, EscapeXML(title), head, bodyAttrs, body)
}

func renderEPUBSection(doc Document, section DocumentSection, options types.EPUBOptions) string {
	profile := documentEPUBProfile(doc)
	var body strings.Builder
	if profile.Three {
		body.WriteString("    <section epub:type=\"")
		body.WriteString(epubSectionType(section.Role))
		body.WriteString("\">\n")
	} else {
		// XHTML 1.1 has no <section> and no epub:type. The role survives as a
		// class, which the stylesheet reads either way.
		body.WriteString("    <div class=\"section ")
		body.WriteString(epubSectionType(section.Role))
		body.WriteString("\">\n")
	}
	if strings.TrimSpace(section.Title) != "" {
		fmt.Fprintf(&body, "      <h1>%s</h1>\n", EscapeXML(section.Title))
	}
	if strings.TrimSpace(section.Subtitle) != "" {
		fmt.Fprintf(&body, "      <p class=\"subtitle\">%s</p>\n", EscapeXML(section.Subtitle))
	}

	listOpen := false
	listOrdered := false
	firstParagraph := true
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
			fmt.Fprintf(&body, "        <li%s>%s</li>\n", epubAlignmentClass(block.Alignment), renderEPUBRuns(block.Runs, profile))
			continue
		}
		closeList()
		// The first paragraph of a section is the one a drop cap belongs to.
		// Only this loop knows which that is, so it marks it and the
		// stylesheet decides whether to do anything about it.
		opening := firstParagraph && block.Kind == BlockParagraph
		if opening {
			firstParagraph = false
		}
		renderEPUBBlock(&body, block, options, profile, opening)
	}
	closeList()
	if profile.Three {
		body.WriteString("    </section>\n")
	} else {
		body.WriteString("    </div>\n")
	}

	bodyAttrs := fmt.Sprintf(" class=\"%s %s\"", epubRoleClass(section.Role), EscapeXML(options.ChapterStyle))
	return epubDocumentShell(profile, doc.Language, displaySectionTitle(section.Title),
		bodyAttrs, `<link rel="stylesheet" type="text/css" href="../styles/book.css"/>`,
		body.String()+"  ")
}

func renderEPUBBlock(out *strings.Builder, block DocumentBlock, options types.EPUBOptions, profile epubProfile, opening bool) {
	content := renderEPUBRuns(block.Runs, profile)
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
			fmt.Fprintf(out, "      <div class=\"scene-break scene-space\"%s></div>\n", epubHidden(profile))
		default:
			fmt.Fprintf(out, "      <div class=\"scene-break scene-asterism\"%s>⁂</div>\n", epubHidden(profile))
		}
	default:
		if opening && options.DropCap {
			align = withEPUBClass(align, "opening")
		}
		fmt.Fprintf(out, "      <p%s>%s</p>\n", align, content)
	}
}

// withEPUBClass adds a class to an attribute that may already carry one, so a
// centred opening paragraph keeps its alignment and gains its drop cap.
func withEPUBClass(attr, name string) string {
	if attr == "" {
		return " class=\"" + name + "\""
	}
	return strings.TrimSuffix(attr, "\"") + " " + name + "\""
}

// epubHidden is the ARIA attribute that tells a screen reader to skip a
// decorative scene break. WAI-ARIA attributes are part of XHTML5 and are not
// in the XHTML 1.1 document type an OPS 2.0.1 document is validated against,
// so an EPUB 2 file leaves them off rather than failing validation over an
// ornament.
func epubHidden(profile epubProfile) string {
	if profile.Three {
		return ` aria-hidden="true"`
	}
	return ""
}

func renderEPUBRuns(runs []DocumentRun, profile epubProfile) string {
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
			// <s> was dropped from XHTML 1.1, so an EPUB 2 file strikes text
			// the same way it underlines it: with a class the stylesheet knows.
			if profile.Three {
				text = "<s>" + text + "</s>"
			} else {
				text = "<span class=\"strike\">" + text + "</span>"
			}
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
		if href := safeExportHref(run.Href); href != "" {
			text = "<a href=\"" + EscapeXML(href) + "\">" + text + "</a>"
		}
		out.WriteString(text)
	}
	return out.String()
}

func safeExportHref(raw string) string {
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
section, div.section { max-width: 42em; margin: 0 auto; }
h1 { text-align: center; break-after: avoid; page-break-after: avoid; }
.classic h1 { margin: 22vh 0 2.5em; }
.minimal h1 { margin: 1.5em 0 1.75em; text-align: left; }
.subtitle { text-align: center; font-style: italic; text-indent: 0 !important; }
blockquote { margin: 1em 1.5em; }
pre { white-space: pre-wrap; font-family: monospace; }
.underline { text-decoration: underline; }
.strike { text-decoration: line-through; }
.align-left { text-align: left !important; }
.align-center { text-align: center !important; }
.align-right { text-align: right !important; }
.align-justify { text-align: justify !important; }
.scene-break { break-inside: avoid; page-break-inside: avoid; text-align: center; }
.scene-asterism { margin: 1.5em 0; }
.scene-rule { border: 0; border-top: 1px solid currentColor; margin: 1.75em auto; width: 18%; }
.scene-space { height: 2em; }
.scene-break + p, h1 + p, h2 + p, h3 + p, blockquote + p { text-indent: 0; }
nav ol, .toc ol { padding-left: 1.5em; }
nav li, .toc li { margin: 0.35em 0; }
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
	// The drop cap. ::first-letter is ordinary CSS that reading systems
	// including Kindle honour; a reader that ignores it simply shows an
	// ordinary paragraph, which is why this is safe to offer. The opening
	// paragraph is never indented — an indent under a raised initial is the
	// mark of a book nobody set.
	if options.DropCap {
		css.WriteString(`p.opening { text-indent: 0; }
p.opening::first-letter { float: left; font-size: 3.2em; line-height: 0.82; padding: 0.02em 0.08em 0 0; font-weight: 700; }
`)
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
