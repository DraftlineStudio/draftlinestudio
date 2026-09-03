//go:build windows

package readaloud

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"draftline/internal/platform"
)

// MemorySnapshot reports process RSS for every WebView2 worker process plus
// this process, for the Read Aloud memory diagnostics. Task-manager-level
// granularity is exactly what's needed: the leak under investigation shows
// up as multi-GB working sets on msedgewebview2.exe workers.
func MemorySnapshot() []string {
	lines := []string{}
	for _, image := range []string{"msedgewebview2.exe", executableName()} {
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+image, "/FO", "CSV", "/NH")
		platform.HideWindow(cmd)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		reader := csv.NewReader(strings.NewReader(string(out)))
		reader.FieldsPerRecord = -1
		records, err := reader.ReadAll()
		if err != nil {
			continue
		}
		var total int64
		count := 0
		for _, record := range records {
			if len(record) < 5 {
				continue
			}
			kb := parseKB(record[4])
			if kb <= 0 {
				continue
			}
			total += kb
			count++
		}
		if count > 0 {
			lines = append(lines, fmt.Sprintf("rss: %s ×%d = %d MB", image, count, total/1024))
		}
	}
	return lines
}

func executableName() string {
	exe, err := os.Executable()
	if err != nil {
		return "draftline.exe"
	}
	parts := strings.Split(strings.ReplaceAll(exe, "\\", "/"), "/")
	return parts[len(parts)-1]
}

func parseKB(field string) int64 {
	cleaned := strings.NewReplacer(",", "", ".", "", " ", "", " K", "", " ", "").Replace(field)
	var kb int64
	if _, err := fmt.Sscanf(cleaned, "%d", &kb); err != nil {
		return 0
	}
	return kb
}
