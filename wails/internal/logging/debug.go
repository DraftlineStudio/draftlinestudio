// Package logging provides debug logging for AI operations.
//
// Content logging is opt-in: nothing is written unless SetEnabled(true) has
// been called (wired to the ai_debug_logging setting). Log files can contain
// manuscript text and prompts, so they are created user-only (0600/0700 on
// Unix; on Windows %AppData% already carries user-only ACLs).
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

var (
	enabled      atomic.Bool
	initOnce     sync.Once
	debugLogger  *log.Logger
	debugLogFile *os.File
)

// SetEnabled turns AI debug logging on or off. Off by default.
func SetEnabled(on bool) {
	enabled.Store(on)
}

// Init initializes the debug log file with rotation.
// Logs are written to AppData/Draftline/logs/ai-*.log
func Init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(configDir, "draftline", "logs")
	_ = os.MkdirAll(logDir, 0700)

	// Rotate logs: keep only last 5
	files, _ := filepath.Glob(filepath.Join(logDir, "ai-*.log"))
	if len(files) >= 5 {
		for i := 0; i < len(files)-4; i++ {
			_ = os.Remove(files[i])
		}
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("ai-%s.log", time.Now().Format("2006-01-02-150405")))
	debugLogFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	debugLogger = log.New(debugLogFile, "", log.LstdFlags)
	debugLogger.Println("=== Draftline AI Debug Log Started ===")
}

// AI logs a formatted message to the AI debug log. No-op unless enabled.
func AI(format string, args ...interface{}) {
	if !enabled.Load() {
		return
	}
	initOnce.Do(Init)
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// AIContent logs content with a label, truncating if necessary. No-op unless
// enabled.
func AIContent(label, content string) {
	if !enabled.Load() {
		return
	}
	if len(content) > 2000 {
		AI("[%s] (truncated to 2000 chars)\n%s\n---END %s---", label, content[:2000], label)
	} else {
		AI("[%s]\n%s\n---END %s---", label, content, label)
	}
}
