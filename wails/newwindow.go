package main

import (
	"os"
	"os/exec"

	"draftline/internal/platform"
	"draftline/internal/types"
)

// Opening a second window.
//
// Draftline is multi-instance by design: two windows with two different books
// is fine, and only the same .draftline being open twice is refused — by
// internal/instancelock, per book, not per application. So a new window is a
// new process of this same executable, started detached so that closing the
// window it was launched from does not take it with it.
//
// It carries no arguments: a fresh window opens on the start screen, which is
// where a book is chosen. That is deliberate — the alternative is a window
// that inherits a book already open somewhere else, which is the one thing the
// lock exists to prevent.

// OpenNewWindow starts another Draftline window on the start screen.
func (a *App) OpenNewWindow() types.RevealResult {
	self, err := os.Executable()
	if err != nil {
		return types.RevealResult{Error: "Draftline could not find its own program file."}
	}
	command := exec.Command(self)
	// Detach only, so the new window outlives the one that opened it.
	//
	// Deliberately NOT platform.HideWindow: that sets STARTF_USESHOWWINDOW
	// with SW_HIDE in the child's STARTUPINFO, which is right for the console
	// helpers it was written for and wrong here. Draftline is a GUI binary, so
	// the child inherits "start hidden" and the new window never appears -- the
	// process starts, reports success, and nothing shows. There is no console
	// to flash in the first place.
	platform.Detach(command)
	if err := command.Start(); err != nil {
		return types.RevealResult{Error: "That window could not be opened."}
	}
	// The child is deliberately not waited on. Reaping it would tie the two
	// windows together, which is the opposite of what a new window is for.
	go func() { _ = command.Wait() }()
	return types.RevealResult{Success: true}
}
