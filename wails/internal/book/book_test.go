package book

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"draftline/internal/types"
)

func testBook() types.BookData {
	return types.BookData{
		Version: "2.0",
		Metadata: types.Metadata{
			Title:  "Test Book",
			Author: "Tester",
		},
		Copyright: "<p>© test</p>",
		Body: []types.ChapterItem{
			{Title: "Chapter One", Type: "chapter", Content: "<p>Hello — “world”.</p>"},
			{Title: "Chapter Two", Type: "chapter", Content: "<p>More text.</p>"},
		},
		FrontMatter: []types.ChapterItem{{Title: "Dedication", Type: "dedication", Content: "<p>For x.</p>"}},
		BackMatter:  []types.ChapterItem{},
		StoryBible:  types.StoryBible{Characters: []types.Character{}},
	}
}

// redirect backups away from the real user config dir
func isolateConfigDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)
}

func TestWriteOpenRoundTrip(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	res := Write(path, testBook(), "test-version")
	if !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}

	// The output must be a valid zip containing manifest.json.
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("output is not a valid zip: %v", err)
	}
	if _, err := ReadZipEntry(r, "manifest.json"); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}
	_ = r.Close()

	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got.Metadata.Title != "Test Book" || len(got.Body) != 2 {
		t.Fatalf("round-trip mismatch: title=%q body=%d", got.Metadata.Title, len(got.Body))
	}
	if got.Body[0].Content != "<p>Hello — “world”.</p>" {
		t.Fatalf("content mismatch: %q", got.Body[0].Content)
	}
}

func TestWriteOpenPersistsAnalysisAndCorrections(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "analysis.draftline")
	b := testBook()
	b.IsIndexed = true
	b.LastIndexed = "2026-08-30T12:00:00-05:00"
	b.Analysis = types.AnalysisData{
		Version: 3,
		EntityResolution: &types.EntityData{
			Version:    1,
			Mentions:   []types.MentionRecord{{ID: "m-0-0", Text: "Mara", Chapter: 0, CharOffset: 3}},
			Entities:   []types.EntityRecord{{ID: "entity-1", Canonical: "Mara", MentionIDs: []string{"m-0-0"}}},
			MergeRules: []types.MergeRule{{Name1: "Mara Voss", Name2: "Mara Ionescu"}},
		},
		Relationships: &types.RelationshipData{
			Version: 1,
			Events:  []types.CharacterEvent{{ID: "manual-1", CharacterIDs: []string{"entity-1"}, Description: "Pinned event"}},
		},
		Story: &types.StoryAnalysisData{
			Version: 1, Engine: "prose-v3", ContentHash: "abc123",
			Chapters: []types.ChapterAnalysis{{ChapterID: "chapter-1", Title: "Chapter One", WordCount: 42}},
		},
		Evidence: &types.EvidenceData{
			Version: 1, Engine: "prose-v3-evidence-v1", ContentHash: "evidence123",
			Records: []types.EvidenceRecord{{
				ID: "evidence-1", Kind: "event", EvidenceType: "discovery", ChapterID: "chapter-1",
				Text: "Mara found the door.", Status: "confirmed", Source: "auto",
			}},
		},
		Fingerprint: &types.StoryFingerprint{
			Version: 2, Engine: "draftline-story-fingerprint-v2", ContentHash: "fingerprint123",
			Events:      []types.FingerprintEvent{{ID: "event-1", Summary: "Mara finds the door", EvidenceIDs: []string{"evidence-1"}}},
			AuthorModel: types.StoryAuthorModel{Canon: []types.CanonRule{{ID: "canon-1", Subject: "Mara", Predicate: "role", Object: "captain"}}},
		},
	}

	if res := Write(path, b, "test-version"); !res.Success {
		t.Fatalf("Write failed: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got.Version != "2.2" || !got.IsIndexed {
		t.Fatalf("expected indexed v2.2 book, got version=%q indexed=%v", got.Version, got.IsIndexed)
	}
	if got.Analysis.EntityResolution == nil || len(got.Analysis.EntityResolution.MergeRules) != 1 {
		t.Fatal("entity analysis or merge rules did not survive round trip")
	}
	if got.Analysis.Relationships == nil || len(got.Analysis.Relationships.Events) != 1 {
		t.Fatal("relationship analysis or manual events did not survive round trip")
	}
	if got.Analysis.Story == nil || got.Analysis.Story.ContentHash != "abc123" || len(got.Analysis.Story.Chapters) != 1 {
		t.Fatal("story analysis did not survive round trip")
	}
	if got.Analysis.Evidence == nil || got.Analysis.Evidence.ContentHash != "evidence123" || len(got.Analysis.Evidence.Records) != 1 || got.Analysis.Evidence.Records[0].Status != "confirmed" {
		t.Fatal("fact/event evidence did not survive round trip")
	}
	if got.Analysis.Fingerprint == nil || got.Analysis.Fingerprint.ContentHash != "fingerprint123" || len(got.Analysis.Fingerprint.Events) != 1 || len(got.Analysis.Fingerprint.AuthorModel.Canon) != 1 {
		t.Fatal("story fingerprint or author canon did not survive round trip")
	}
}

func TestChapterHistoryRoundTripDeduplicatesAndSurvivesNormalSave(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "history.draftline")
	b := testBook()
	EnsureBookChapterIDs(&b)
	chapterID := b.Body[0].ID

	if res := Write(path, b, "v1"); !res.Success {
		t.Fatalf("initial write: %s", res.Error)
	}
	request := types.ChapterSnapshotRequest{
		ChapterID: chapterID, Section: "body", ChapterTitle: b.Body[0].Title,
		Content: b.Body[0].Content, Reason: "Writing session",
	}
	if res := WriteWithSnapshots(path, b, "v1", []types.ChapterSnapshotRequest{request, request}); !res.Success {
		t.Fatalf("snapshot write: %s", res.Error)
	}
	entries, err := ListChapterHistory(path, chapterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("duplicate content should create one snapshot, got %d", len(entries))
	}
	snapshot, err := GetChapterHistorySnapshot(path, entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Content != b.Body[0].Content || snapshot.Entry.WordCount != 2 {
		t.Fatalf("unexpected snapshot: words=%d content=%q", snapshot.Entry.WordCount, snapshot.Content)
	}

	b.Metadata.Title = "Saved Again"
	if res := Write(path, b, "v1"); !res.Success {
		t.Fatalf("normal save: %s", res.Error)
	}
	entries, err = ListChapterHistory(path, chapterID)
	if err != nil || len(entries) != 1 {
		t.Fatalf("normal save must preserve history: entries=%d err=%v", len(entries), err)
	}
}

func TestLegacyChapterIDsAreAssignedAndPersisted(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "ids.draftline")
	if res := Write(path, testBook(), "v1"); !res.Success {
		t.Fatalf("write: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body[0].ID == "" || got.Body[0].ID == got.Body[1].ID {
		t.Fatalf("chapter IDs were not assigned uniquely: %#v", got.Body)
	}
}

func TestChapterHistoryDeduplicatesAgainstLatestSnapshot(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "latest-history.draftline")
	b := testBook()
	EnsureBookChapterIDs(&b)
	chapterID := b.Body[0].ID
	if res := Write(path, b, "v1"); !res.Success {
		t.Fatal(res.Error)
	}
	for _, content := range []string{"<p>Version A</p>", "<p>Version B</p>", "<p>Version B</p>"} {
		request := types.ChapterSnapshotRequest{ChapterID: chapterID, Section: "body", ChapterTitle: "Chapter One", Content: content}
		if res := WriteWithSnapshots(path, b, "v1", []types.ChapterSnapshotRequest{request}); !res.Success {
			t.Fatal(res.Error)
		}
	}
	entries, err := ListChapterHistory(path, chapterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("latest duplicate should be skipped, got %d entries", len(entries))
	}
}

func TestOpenLegacyIndexedBookWithoutAnalysisRequiresReindex(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "legacy.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	mw, _ := w.Create("manifest.json")
	_, _ = mw.Write([]byte(`{"version":"2.0","is_indexed":true,"last_indexed":"stale","body":[]}`))
	_ = w.Close()
	_ = f.Close()

	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got.IsIndexed || got.LastIndexed != "" {
		t.Fatal("legacy book without persisted analysis must require re-indexing")
	}
}

func TestWriteReplacesExistingFileAtomically(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "book.draftline")

	if res := Write(path, testBook(), "v1"); !res.Success {
		t.Fatalf("first write: %s", res.Error)
	}
	b := testBook()
	b.Metadata.Title = "Second Save"
	if res := Write(path, b, "v1"); !res.Success {
		t.Fatalf("second write: %s", res.Error)
	}
	got, err := Open(path)
	if err != nil {
		t.Fatalf("Open after overwrite: %v", err)
	}
	if got.Metadata.Title != "Second Save" {
		t.Fatalf("expected updated title, got %q", got.Metadata.Title)
	}
	// No stray temp files next to the book.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Fatalf("unexpected file left beside book: %s", e.Name())
		}
	}
}

func TestOpenFailsOnMissingChapterEntry(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "missing.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	mw, _ := w.Create("manifest.json")
	// Manifest references body/000.html but the entry is never written.
	_, _ = mw.Write([]byte(`{"version":"2.0","body":[{"title":"Chapter One","type":"chapter","file":"body/000.html"}]}`))
	_ = w.Close()
	_ = f.Close()

	got, err := Open(path)
	if err == nil {
		t.Fatalf("expected error for missing chapter entry, got body=%d", len(got.Body))
	}
	if !strings.Contains(err.Error(), "Chapter One") || !strings.Contains(err.Error(), "body/000.html") {
		t.Fatalf("error should name the chapter and file, got: %v", err)
	}
}

func TestOpenFailsOnOversizedChapterEntry(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "oversized.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	mw, _ := w.Create("manifest.json")
	_, _ = mw.Write([]byte(`{"version":"2.0","body":[{"title":"Big Chapter","type":"chapter","file":"body/000.html"}]}`))
	bw, _ := w.Create("body/000.html")
	// Highly compressible but exceeds MaxEntrySize (50 MB) when decompressed.
	_, _ = bw.Write(make([]byte, (50<<20)+1))
	_ = w.Close()
	_ = f.Close()

	if _, err := Open(path); err == nil {
		t.Fatal("expected error for oversized chapter entry, not an empty chapter")
	}
}

func TestOpenRejectsManifestReferenceAmplification(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "amplified.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)

	// A single valid body entry, referenced an absurd number of times.
	bw, _ := w.Create("body/000.html")
	_, _ = bw.Write([]byte("<p>chapter</p>"))

	var b strings.Builder
	b.WriteString(`{"version":"2.0","body":[`)
	refs := MaxManifestRefs + 1
	for i := 0; i < refs; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"title":"C","type":"chapter","file":"body/000.html"}`)
	}
	b.WriteString(`]}`)

	mw, _ := w.Create("manifest.json")
	_, _ = mw.Write([]byte(b.String()))
	_ = w.Close()
	_ = f.Close()

	_, err = Open(path)
	if err == nil {
		t.Fatal("expected rejection of manifest with excessive references")
	}
	if !strings.Contains(err.Error(), "limit") {
		t.Fatalf("error should mention the reference limit, got: %v", err)
	}
}

func TestOpenRejectsZipBomb(t *testing.T) {
	isolateConfigDir(t)
	path := filepath.Join(t.TempDir(), "bomb.draftline")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	mw, _ := w.Create("manifest.json")
	_, _ = mw.Write([]byte(`{"version":"2.0"}`))
	bw, _ := w.Create("body/000.html")
	_, _ = bw.Write(make([]byte, 20<<20)) // 20MB zeros: extreme compression ratio
	_ = w.Close()
	_ = f.Close()

	if _, err := Open(path); err == nil {
		t.Fatal("expected zip bomb to be rejected")
	}
}
