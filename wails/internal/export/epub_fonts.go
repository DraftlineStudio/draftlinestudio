package export

import (
	"fmt"
	"strings"
)

func epubFontEntries(family string) []epubEntry {
	font, ok := embeddedPDFFonts[family]
	if !ok {
		return nil
	}
	base := "OEBPS/fonts/"
	return []epubEntry{
		{Name: base + family + "-regular.ttf", Data: font.Regular},
		{Name: base + family + "-bold.ttf", Data: font.Bold},
		{Name: base + family + "-italic.ttf", Data: font.Italic},
		{Name: base + family + "-bold-italic.ttf", Data: font.BoldItalic},
	}
}

func epubFontManifest(family string) string {
	if _, ok := embeddedPDFFonts[family]; !ok {
		return ""
	}
	var out strings.Builder
	for _, style := range []string{"regular", "bold", "italic", "bold-italic"} {
		fmt.Fprintf(&out, "    <item id=\"font-%s\" href=\"fonts/%s-%s.ttf\" media-type=\"font/ttf\"/>\n", style, family, style)
	}
	return out.String()
}
