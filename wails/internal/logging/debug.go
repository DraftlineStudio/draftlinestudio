// Package logging provides debug logging for AI operations.
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var debugLogger *log.Logger
var debugLogFile *os.File

// Init initializes the debug log file with rotation.
// Logs are written to AppData/Draftline/logs/ai-*.log
func Init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(configDir, "draftline", "logs")
	_ = os.MkdirAll(logDir, 0755)

	// Rotate logs: keep only last 5
	files, _ := filepath.Glob(filepath.Join(logDir, "ai-*.log"))
	if len(files) >= 5 {
		for i := 0; i < len(files)-4; i++ {
			_ = os.Remove(files[i])
		}
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("ai-%s.log", time.Now().Format("2006-01-02-150405")))
	debugLogFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	debugLogger = log.New(debugLogFile, "", log.LstdFlags)
	debugLogger.Println("=== Draftline AI Debug Log Started ===")
}

// AI logs a formatted message to the AI debug log.
func AI(format string, args ...interface{}) {
	if debugLogger == nil {
		Init()
	}
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// AIContent logs content with a label, truncating if necessary.
func AIContent(label, content string) {
	if len(content) > 2000 {
		AI("[%s] (truncated to 2000 chars)\n%s\n---END %s---", label, content[:2000], label)
	} else {
		AI("[%s]\n%s\n---END %s---", label, content, label)
	}
}
