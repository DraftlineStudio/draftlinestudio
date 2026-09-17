package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
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
// Addressed by a key derived from the path, never by the path itself. The
// webview only ever repeats a key this process gave it, and a key that matches
// no recent project is a 404 -- so no path crosses the boundary and there is
// nothing to traverse. A handler taking a path would be a handler that reads
// any file on the machine on request, which is not a trade worth making for a
// thumbnail.
//
// It was addressed by POSITION first, and that was wrong: opening a book moves
// it to the front of the recents list, so a position meant one book to the
// start screen and a different one to this handler, and every book without art
// briefly wore the art of whichever had taken its place. A key is stable while
// the path is, which is exactly as long as it needs to be.

const recentCoverPrefix = "/recent-cover/"

// recentCovers serves the cover thumbnail of a recent project, by its key.
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
		key := strings.TrimSuffix(rest, "/")
		if key == "" {
			http.NotFound(w, r)
			return
		}
		archive := ""
		for _, recent := range a.GetRecentProjects() {
			if recent.CoverKey == key {
				archive = recent.Path
				break
			}
		}
		if archive == "" {
			http.NotFound(w, r)
			return
		}

		data, file, err := newestCoverThumb(archive)
		if err != nil || data == nil {
			// A book with no editions, or none with artwork, is the ordinary
			// case rather than a fault. The start screen falls back to its
			// colour on a 404.
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", coverContentType(file))
		// The key is stable but what it points at is not: attaching new artwork
		// to an edition changes the bytes behind the same address.
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

// recentCoverKey is the address a project's cover art is served from.
//
// A truncated SHA-256 of the path: stable for as long as the path is, and
// not the path, which is the only property that matters here. Collisions do
// not need guarding against -- two books would have to share 64 bits of
// digest to show each other's covers on one start screen.
func recentCoverKey(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:8])
}
