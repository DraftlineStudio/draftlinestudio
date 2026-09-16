package export

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestBundleLaysOutOneFolderPerFormat(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Wide Water — First edition.zip")
	root := "Wide Water — First edition"

	wrap := filepath.Join(dir, "paperback-wrap.pdf")
	if err := os.WriteFile(wrap, []byte("%PDF-1.7 wrap"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Bundle(target, []BundleMember{
		{Path: root + "/Paperback/Wide Water — print interior.pdf", Data: []byte("%PDF-1.7 interior")},
		{Path: root + "/Paperback/paperback-wrap.pdf", Source: wrap},
		{Path: root + "/eBook/Wide Water.epub", Data: []byte("PK epub")},
		{Path: root + "/eBook/cover.jpg", Data: []byte("\xff\xd8\xff cover")},
	})
	if !result.Success {
		t.Fatalf("bundle failed: %s", result.Error)
	}
	if result.FilePath != target {
		t.Fatalf("wrote %q, wanted %q", result.FilePath, target)
	}

	r, err := zip.OpenReader(target)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()

	got := map[string]bool{}
	for _, f := range r.File {
		got[f.Name] = true
	}
	for _, want := range []string{
		root + "/Paperback/Wide Water — print interior.pdf",
		root + "/Paperback/paperback-wrap.pdf",
		root + "/eBook/Wide Water.epub",
		root + "/eBook/cover.jpg",
	} {
		if !got[want] {
			t.Errorf("missing %q from the bundle", want)
		}
	}
}

// An artwork file that has moved since it was attached must fail the whole
// bundle rather than producing a package that is quietly missing its cover.
func TestBundleRefusesWhenSourceArtworkIsGone(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bundle.zip")
	result := Bundle(target, []BundleMember{
		{Path: "book/Paperback/interior.pdf", Data: []byte("pdf")},
		{Path: "book/Paperback/wrap.tif", Source: filepath.Join(dir, "not-here.tif")},
	})
	if result.Success {
		t.Fatal("a bundle with unreadable artwork reported success")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("a failed bundle left a file at the destination")
	}
}

func TestBundleRefusesAnEmptyRequest(t *testing.T) {
	if Bundle(filepath.Join(t.TempDir(), "x.zip"), nil).Success {
		t.Fatal("an empty bundle reported success")
	}
}

func TestSafeNameKeepsATitleUsableAsAFolder(t *testing.T) {
	cases := map[string]string{
		"Wide Water — First edition": "Wide Water — First edition",
		"Hope: A Novel":             "Hope- A Novel",
		"  spaced   out  ":          "spaced out",
		"trailing dot.":             "trailing dot",
		"a/b\\c":                    "a-b-c",
		"":                          "Untitled",
		"Sci-fi Collection":         "Sci-fi Collection",
	}
	for in, want := range cases {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, wanted %q", in, got, want)
		}
	}
}

func TestUniqueFolderKeepsTwoOfTheSameFormatApart(t *testing.T) {
	taken := map[string]bool{}
	if got := UniqueFolder(taken, "Paperback"); got != "Paperback" {
		t.Fatalf("first = %q", got)
	}
	if got := UniqueFolder(taken, "Paperback"); got != "Paperback (2)" {
		t.Fatalf("second = %q", got)
	}
	if got := UniqueFolder(taken, "paperback"); got != "paperback (3)" {
		t.Fatalf("third = %q", got)
	}
}
