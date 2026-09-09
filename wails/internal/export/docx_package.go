package export

import (
	"fmt"
	"strings"
)

type docxPart struct {
	Name string
	Data []byte
}

func buildDOCXParts(doc Document, links map[string]string) []docxPart {
	return []docxPart{
		{Name: "[Content_Types].xml", Data: []byte(docxContentTypes)},
		{Name: "_rels/.rels", Data: []byte(docxRootRels)},
		{Name: "docProps/core.xml", Data: []byte(renderDOCXCore(doc))},
		{Name: "docProps/app.xml", Data: []byte(docxAppProperties)},
		{Name: "word/styles.xml", Data: []byte(docxStyles)},
		{Name: "word/numbering.xml", Data: []byte(renderDOCXNumbering())},
		{Name: "word/_rels/document.xml.rels", Data: []byte(renderDOCXRelationships(links))},
		{Name: "word/document.xml", Data: []byte(renderDOCXDocument(doc, links))},
	}
}

const docxContentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`

const docxRootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`

func renderDOCXCore(doc Document) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <dc:title>%s</dc:title><dc:creator>%s</dc:creator>
</cp:coreProperties>`, EscapeXML(doc.Title), EscapeXML(doc.Author))
}

const docxAppProperties = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>Draftline</Application></Properties>`

func renderDOCXRelationships(links map[string]string) string {
	var out strings.Builder
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
  <Relationship Id="rIdNumbering" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>
`)
	for href, id := range links {
		fmt.Fprintf(&out, "  <Relationship Id=\"%s\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink\" Target=\"%s\" TargetMode=\"External\"/>\n", id, EscapeXML(href))
	}
	out.WriteString("</Relationships>")
	return out.String()
}

const docxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:pPr><w:spacing w:after="160" w:line="276" w:lineRule="auto"/></w:pPr><w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos"/><w:sz w:val="24"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:pPr><w:jc w:val="center"/><w:spacing w:after="360"/></w:pPr><w:rPr><w:b/><w:sz w:val="72"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:pPr><w:jc w:val="center"/><w:spacing w:after="240"/></w:pPr><w:rPr><w:i/><w:sz w:val="28"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="Heading 1"/><w:pPr><w:keepNext/><w:spacing w:before="480" w:after="240"/></w:pPr><w:rPr><w:b/><w:sz w:val="48"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="Heading 2"/><w:pPr><w:keepNext/><w:spacing w:before="300" w:after="160"/></w:pPr><w:rPr><w:b/><w:sz w:val="34"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:pPr><w:ind w:left="720" w:right="720"/><w:spacing w:before="120" w:after="120"/></w:pPr><w:rPr><w:i/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:pPr><w:ind w:left="360"/><w:spacing w:before="120" w:after="120"/></w:pPr><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/><w:sz w:val="20"/></w:rPr></w:style>
</w:styles>`

func renderDOCXNumbering() string {
	var out strings.Builder
	out.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="0"><w:multiLevelType w:val="multilevel"/>`)
	for level := 0; level < 9; level++ {
		left := 720 + level*360
		fmt.Fprintf(&out, `<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/><w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr></w:lvl>`, level, left)
	}
	out.WriteString(`</w:abstractNum>
  <w:abstractNum w:abstractNumId="1"><w:multiLevelType w:val="multilevel"/>`)
	for level := 0; level < 9; level++ {
		left := 720 + level*360
		fmt.Fprintf(&out, `<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%%%d."/><w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr></w:lvl>`, level, level+1, left)
	}
	out.WriteString(`</w:abstractNum>
  <w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num><w:num w:numId="2"><w:abstractNumId w:val="1"/></w:num>
</w:numbering>`)
	return out.String()
}
