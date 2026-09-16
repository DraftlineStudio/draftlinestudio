package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"draftline/internal/platform"
	"draftline/internal/types"
)

// Showing a file where it lives.
//
// A print-ready original is not kept inside the project — it is the author's
// own file, often hundreds of megabytes, and the record remembers where it is
// rather than copying it. That makes "where is it?" a real question, and the
// honest answer is the one the operating system already knows how to give:
// open the folder with the file selected.
//
// Every platform does this differently and two of the three take the file
// path rather than the folder, so the selection is preserved where it can be.
// Linux has no standard "select this file" call across file managers, so it
// opens the containing folder, which is the same answer one step coarser.

// RevealInFileManager opens the system file browser on the folder holding
// path, with the file selected where the platform supports it.
func (a *App) RevealInFileManager(path string) types.RevealResult {
	target := strings.TrimSpace(path)
	if target == "" {
		return types.RevealResult{Error: "Draftline does not know where that file is."}
	}
	info, err := os.Stat(target)
	if err != nil {
		// The file has moved or been deleted since it was attached. Falling
		// back to its folder is still useful — that is usually where the
		// author went looking — but only when the folder itself is there.
		dir := filepath.Dir(target)
		if _, dirErr := os.Stat(dir); dirErr != nil {
			return types.RevealResult{Error: "That file is no longer where Draftline recorded it."}
		}
		if err := openFolder(dir); err != nil {
			return types.RevealResult{Error: "That folder could not be opened."}
		}
		return types.RevealResult{Success: true, Note: "That file has moved. Draftline opened the folder it was recorded in."}
	}
	if info.IsDir() {
		if err := openFolder(target); err != nil {
			return types.RevealResult{Error: "That folder could not be opened."}
		}
		return types.RevealResult{Success: true}
	}
	if err := revealFile(target); err != nil {
		return types.RevealResult{Error: "That file could not be shown."}
	}
	return types.RevealResult{Success: true}
}

func revealFile(path string) error {
	switch runtime.GOOS {
	case "windows":
		// explorer returns a non-zero exit code even when it succeeds, so the
		// result of Run is deliberately not the thing that decides.
		command := exec.Command("explorer", "/select,"+filepath.Clean(path))
		platform.HideWindow(command)
		_ = command.Run()
		return nil
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return openFolder(filepath.Dir(path))
	}
}

func openFolder(dir string) error {
	switch runtime.GOOS {
	case "windows":
		command := exec.Command("explorer", filepath.Clean(dir))
		platform.HideWindow(command)
		_ = command.Run()
		return nil
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		command := exec.Command("xdg-open", dir)
		platform.Detach(command)
		if err := command.Start(); err != nil {
			return fmt.Errorf("xdg-open: %w", err)
		}
		return nil
	}
}
