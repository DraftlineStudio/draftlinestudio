package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"draftline/internal/types"
)

// The cover art on the start screen.
//
// A recent project is a path and nothing else, so its artwork has to be read
// out of the archive before the book is open. That is cheap: an edition stores
// a small prepared thumbnail beside the full-size art, and this reads only
// that one member.
//
// Addressed by position in the recents list, never by path. The webview asks
// for /recent-cover/0 and the answer comes from whatever is first in the list
// this process holds, so no path crosses the boundary and there is nothing to
// traverse. A handler taking a path would be a handler that reads any file on
// the machine on request, which is not a trade worth making for a thumbnail.

const recentCoverPrefix = "/recent-cover/"

// recentCovers serves the cover thumbnail of a recent project, by position.
func (a *App) recentCovers() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rest, ok := strings.CutPrefix(r.URL.Path, recentCoverPrefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
		index, err := strconv.Atoi(strings.TrimSuffix(rest, "/"))
		if err != nil || index < 0 {
			http.NotFound(w, r)
			return
		}
		recents := a.GetRecentProjects()
		if index >= len(recents) {
			http.NotFound(w, r)
			return
		}

		data, file, err := newestCoverThumb(recents[index].Path)
		if err != nil || data == nil {
			// A book with no editions, or none with artwork, is the ordinary
			// case rather than a fault. The start screen falls back to its
			// colour on a 404.
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", coverContentType(file))
		// Not cacheable by address: position 0 is a different book as soon as
		// another is opened, and the art behind one edition can be replaced.
		w.Header().Set("Cache-Control", "no-store")
		http.ServeContent(w, r, file, time.Time{}, newByteSeeker(data))
	})
}

// newestCoverThumb reads the thumbnail of the most recently added edition that
// has artwork, returning the bytes and the member name it came from.
//
// "Most recent" is the last edition in the index carrying a cover. The index is
// appended to as editions are registered, so the end of it is the newest — and
// an edition that supersedes another is added after the one it supersedes.
func newestCoverThumb(archivePath string) ([]byte, string, error) {
	if strings.TrimSpace(archivePath) == "" {
		return nil, "", nil
	}
	raw, err := readArchiveMember(archivePath, "editions/index.json")
	if err != nil || raw == nil {
		return nil, "", err
	}
	var index types.EditionIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, "", nil
	}
	for i := len(index.Editions) - 1; i >= 0; i-- {
		edition := index.Editions[i]
		if edition.Cover == nil {
			continue
		}
		file := strings.TrimSpace(edition.Cover.ThumbFile)
		if file == "" || !safeAssetSegment(file) || !safeAssetSegment(edition.ID) {
			continue
		}
		data, err := readArchiveMember(archivePath, "editions/"+edition.ID+"/"+file)
		if err != nil || data == nil {
			// Record without bytes: an edition whose art was removed from the
			// archive by hand. Keep looking rather than give up on the book.
			continue
		}
		return data, file, nil
	}
	return nil, "", nil
}
