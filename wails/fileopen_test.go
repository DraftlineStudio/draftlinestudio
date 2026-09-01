package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchFilePath(t *testing.T) {
	dir := t.TempDir()
	book := filepath.Join(dir, "novel.draftline")
	epub := filepath.Join(dir, "Imported Book.EPUB")
	for _, p := range []string{book, epub} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("picks the first openable existing file, skipping flags", func(t *testing.T) {
		got := launchFilePath([]string{"--flag", "notes.txt", book, epub}, dir)
		if got != book {
			t.Fatalf("got %q want %q", got, book)
		}
	})

	t.Run("extension match is case-insensitive", func(t *testing.T) {
		if got := launchFilePath([]string{epub}, dir); got != epub {
			t.Fatalf("got %q want %q", got, epub)
		}
	})

	t.Run("relative paths resolve against the working directory", func(t *testing.T) {
		got := launchFilePath([]string{"novel.draftline"}, dir)
		if got != book {
			t.Fatalf("got %q want %q", got, book)
		}
	})

	t.Run("missing files and unsupported types yield nothing", func(t *testing.T) {
		if got := launchFilePath([]string{"ghost.draftline", "story.pdf", "notes.txt"}, dir); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("storiverse files are forwarded so the app can explain itself", func(t *testing.T) {
		universe := filepath.Join(dir, "saga.storiverse")
		if err := os.WriteFile(universe, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := launchFilePath([]string{universe}, dir); got != universe {
			t.Fatalf("got %q want %q", got, universe)
		}
	})

	t.Run("a leading executable path never masks the document", func(t *testing.T) {
		// Second-instance args may or may not include argv[0]; either way the
		// exe must be skipped by the extension allowlist, not by position.
		got := launchFilePath([]string{`C:\SRV\draftline\build\bin\draftline.exe`, book}, dir)
		if got != book {
			t.Fatalf("got %q want %q", got, book)
		}
	})
}

func TestTakePendingOpenPathIsOneShot(t *testing.T) {
	setPendingOpenPath("C:/tmp/x.draftline")
	a := &App{}
	if got := a.TakePendingOpenPath(); got != "C:/tmp/x.draftline" {
		t.Fatalf("first take returned %q", got)
	}
	if got := a.TakePendingOpenPath(); got != "" {
		t.Fatalf("second take should be empty, got %q", got)
	}
}
