package indexing

import (
	"testing"

	"github.com/jdkato/prose/v3"
)

func TestProbeProseEntities(t *testing.T) {
	text := `The detective crossed Chicago at Hubbard Street. He turned onto Ontario at Wells Ave and followed the river north toward Lower Wacker Drive. Detective Daniel Hanlon waited beneath the awning. Hanlon's coat was wet. Mara Ionescu waved to him.`
	doc, err := prose.NewDocument(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, entity := range doc.Entities() {
		t.Logf("entity %q label=%s span=%d:%d", entity.Text, entity.Label, entity.Start, entity.End())
	}
	for _, token := range doc.Tokens() {
		t.Logf("token %q tag=%s label=%s span=%d:%d", token.Text, token.Tag, token.Label, token.Start, token.End())
	}
}

func TestProbeProseFictionalNames(t *testing.T) {
	text := `Mara Ionescu entered the chamber. Captain Glorpashlrop followed Mara. The Echo waited in orbit while FleetCom issued new orders. Veyr studied Qalis beneath the red sun.`
	doc, err := prose.NewDocument(text)
	if err != nil {
		t.Fatal(err)
	}
	for _, entity := range doc.Entities() {
		t.Logf("entity %q label=%s span=%d:%d", entity.Text, entity.Label, entity.Start, entity.End())
	}
	for _, token := range doc.Tokens() {
		t.Logf("token %q tag=%s label=%s", token.Text, token.Tag, token.Label)
	}
}
