package main

// Size guardrail: the 2026-08 technical-debt audit found app.go had silently
// regrown by 800 lines because nothing flagged it. This test makes the doc's
// "800-line review threshold" mechanical: any non-test source file crossing
// the threshold fails the build unless it is on the ratchet allowlist below,
// and an allowlisted file may shrink but never grow past its recorded size.
//
// CSS is deliberately exempt: global.css is an intentional single-file
// stylesheet (user workflow — see its header comment).
//
// When this test fails you have two honest options: split the file, or — with
// a deliberate reason recorded in docs/TECHNICAL-DEBT.md — raise its ratchet
// entry here.

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const debtLineThreshold = 800

// ratchet maps repo-relative paths (forward slashes) to their maximum allowed
// line count. Values are the measured size at 0.16.02468 plus a small margin
// so routine edits don't trip the gate; shrink them as the files shrink.
var debtRatchet = map[string]int{
	"app.go":                          1500, // 1,468 at 02470; analysis orchestration extracted, CLI drivers still here
	"import.go":                       780,  // 722 at 02468; EPUB+DOCX importers (backlog: internal/importer)
	"frontend/src/store/bookStore.ts": 930,  // raised 02490: continuity review decisions (TECHNICAL-DEBT.md); extract decision writers if this needs a 4th raise
	"frontend/src/components/dialogs/ExportWizard.tsx":      860, // 833; oldest open backlog item
	"frontend/src/components/characters/CharactersView.tsx": 840, // 811; five components in one file
	"frontend/src/components/tools/AIStudio/index.tsx":      780, // 748
}

func TestSourceFileSizeGuardrail(t *testing.T) {
	roots := []struct {
		dir  string
		exts []string
	}{
		{".", []string{".go"}},
		{"frontend/src", []string{".ts", ".tsx"}},
	}

	for _, root := range roots {
		err := filepath.WalkDir(root.dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() {
				switch name {
				case "frontend", "node_modules", "dist", "wailsjs", "build", "__tests__":
					// "frontend" is only skipped for the Go root walk; the
					// second root enters frontend/src explicitly.
					if root.dir == "." && name == "frontend" {
						return filepath.SkipDir
					}
					if name != "frontend" {
						return filepath.SkipDir
					}
				}
				return nil
			}
			ext := filepath.Ext(name)
			keep := false
			for _, want := range root.exts {
				if ext == want {
					keep = true
				}
			}
			if !keep || strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".test.tsx") || strings.HasSuffix(name, ".d.ts") {
				return nil
			}

			lines, err := countLines(path)
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(path)
			if limit, ok := debtRatchet[rel]; ok {
				if lines > limit {
					t.Errorf("%s is %d lines, over its ratchet of %d — split it, or record a reason in docs/TECHNICAL-DEBT.md and raise the ratchet", rel, lines, limit)
				}
				return nil
			}
			if lines > debtLineThreshold {
				t.Errorf("%s is %d lines, over the %d-line threshold and not on the ratchet allowlist — split it, or add a deliberate entry with a reason", rel, lines, debtLineThreshold)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root.dir, err)
		}
	}
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	n := 0
	for scanner.Scan() {
		n++
	}
	return n, scanner.Err()
}
