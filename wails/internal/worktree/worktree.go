// Package worktree opens a .draftline as a directory of loose files and packs
// it back into the archive on a schedule.
//
// # WHY THIS EXISTS
//
// A .draftline is a ZIP, and every autosave used to rebuild the whole thing:
// deflate the entire manuscript, write the entire file. That costs processor
// time and battery on a phone for one changed paragraph, and it defeats the
// block-level upload that Dropbox, Drive and OneDrive do — a rewritten ZIP is
// a new file end to end, so the whole book re-uploads every few seconds.
//
// So a book is opened once into a working directory and edited there as loose
// files. One changed chapter writes one small file. The archive is rebuilt
// only on a rhythm, and when it is rebuilt it is written to a temp file beside
// itself and renamed over the original, so a sync client watching the folder
// either sees the old book or the new one and never an archive that is half
// written. That is the same guarantee fsutil.WriteFileAtomic gives every other
// file Draftline writes, applied to the project itself.
//
// # WHERE IT LIVES
//
// Not beside the archive. A working directory next to a book inside a synced
// folder would sync too, and thousands of small file events is a worse problem
// than the one being solved. Working directories live under the application's
// own data directory, one per book, named for the book so a human looking at
// the folder can tell what it is:
//
//	<root>/.The Weather House.draftline-3f9a21c7/
//	    content/       the archive, extracted member for member
//	    session.json   what this working copy is and how it stands
//
// The leading dot hides the directory on macOS and Linux; on Windows the
// application data directory is out of the way already. The eight hex digits
// are the start of a SHA-256 of the archive's absolute path, and they are the
// part that matters: two books can easily be called novel.draftline in two
// different folders, and without the path in the name they would silently
// share one working copy and overwrite each other.
//
// # THE TWO MODES
//
// With autosave on, the working copy is authoritative between repacks: if the
// application dies, the tree holds work the archive does not, and Open returns
// StateRecovered so the caller can pack it straight back out.
//
// With autosave off the archive is authoritative and the tree is scratch. A
// writer who turns autosave off is told their work is discarded unless they
// save; if they do not, the tree is thrown away and re-extracted. Open enforces
// that on the next open too, because "before closing" is not a promise a
// process that was killed can keep.
package worktree

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"draftline/internal/fsutil"
	"draftline/internal/ziputil"
)

// sessionVersion is the on-disk format of session.json. An unrecognised
// version is not read and not written over: the tree is re-extracted instead,
// which costs one unzip and cannot corrupt anything.
const sessionVersion = 1

const (
	contentDir  = "content"
	sessionFile = "session.json"
)

// State is what Open found and what it did about it.
type State int

const (
	// StateExtracted means the archive was unpacked fresh.
	StateExtracted State = iota
	// StateReused means an existing working copy matched the archive and was
	// kept, so opening cost nothing.
	StateReused
	// StateRecovered means the working copy held work the archive did not,
	// from a session that ended without a final repack. The caller should
	// repack promptly.
	StateRecovered
	// StateDiscarded means the working copy held unsaved work from a session
	// with autosave off, and was thrown away as that mode promises.
	StateDiscarded
	// StateConflict means the working copy held unsaved work AND the archive
	// changed underneath it — most likely a sync client brought down a newer
	// copy. Nothing is destroyed and nothing is guessed; the caller must ask.
	StateConflict
)

func (s State) String() string {
	switch s {
	case StateExtracted:
		return "extracted"
	case StateReused:
		return "reused"
	case StateRecovered:
		return "recovered"
	case StateDiscarded:
		return "discarded"
	case StateConflict:
		return "conflict"
	}
	return "unknown"
}

// session is the working copy's own record of itself, written beside the
// content rather than inside it so it never lands in the archive.
type session struct {
	Version int    `json:"version"`
	Archive string `json:"archive"`
	// ArchiveSize and ArchiveModified are how a changed archive is noticed:
	// a sync client replacing the file underneath a working copy has to be
	// caught, or a phone would quietly write its stale copy back over it.
	ArchiveSize     int64     `json:"archive_size"`
	ArchiveModified time.Time `json:"archive_modified"`
	// Autosave records the mode the session was opened in, because what to do
	// with an unsaved working copy depends entirely on it.
	Autosave   bool      `json:"autosave"`
	Dirty      bool      `json:"dirty"`
	OpenedAt   time.Time `json:"opened_at"`
	RepackedAt time.Time `json:"repacked_at,omitempty"`
}

// Tree is one book, open as a directory.
type Tree struct {
	dir     string
	archive string
	state   session
}

// Options configure Open.
type Options struct {
	// Root is the directory working copies live under, normally the
	// application's data directory. It is created if it does not exist.
	Root string
	// Autosave declares the mode this session runs in. See the package
	// comment: it decides what happens to unsaved work.
	Autosave bool
	// Resolve settles a StateConflict. Nil means Open returns StateConflict
	// and an unusable Tree, which is what a UI wants so it can ask.
	Resolve Resolution
}

// Resolution is how a caller answers a conflict between a working copy with
// unsaved work and an archive that changed underneath it.
type Resolution int

const (
	// ResolveAsk is the default: refuse to choose.
	ResolveAsk Resolution = iota
	// ResolveKeepWorkingCopy keeps the unsaved work and will overwrite the
	// archive on the next repack.
	ResolveKeepWorkingCopy
	// ResolveTakeArchive throws the working copy away and re-extracts.
	ResolveTakeArchive
)

// ErrConflict is returned with a nil Tree when a working copy and its archive
// have both moved on and Options.Resolve is ResolveAsk.
var ErrConflict = errors.New("the working copy and the book on disk have both changed")

// DirFor reports the working directory for an archive, without touching disk.
// It is exported so a caller can show a writer where their work is.
func DirFor(root, archivePath string) string {
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		abs = archivePath
	}
	// Windows paths are case-insensitive, so the same book reached by two
	// spellings must hash the same. Lowercasing is wrong for case-sensitive
	// filesystems in principle; in practice a book opened as two different
	// cases on Linux is not a case worth splitting a working copy over.
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(abs))))
	name := filepath.Base(abs)
	return filepath.Join(root, fmt.Sprintf(".%s-%s", name, hex.EncodeToString(sum[:4])))
}

// Open prepares a working copy of archivePath and reports what it had to do.
func Open(archivePath string, opts Options) (*Tree, State, error) {
	if strings.TrimSpace(opts.Root) == "" {
		return nil, StateExtracted, errors.New("worktree: no root directory given")
	}
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		return nil, StateExtracted, fmt.Errorf("worktree: resolve %q: %w", archivePath, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, StateExtracted, fmt.Errorf("worktree: open %q: %w", archivePath, err)
	}

	dir := DirFor(opts.Root, abs)
	tree := &Tree{dir: dir, archive: abs}

	previous, hasPrevious := readSession(dir)
	archiveMoved := hasPrevious &&
		(previous.ArchiveSize != info.Size() || !previous.ArchiveModified.Equal(info.ModTime()))

	state := StateExtracted
	switch {
	case !hasPrevious:
		state = StateExtracted

	case !previous.Dirty:
		// Nothing unsaved. Reuse the tree when the archive is untouched,
		// re-extract when it is not: an unchanged tree is worth nothing and
		// the archive is always right in this branch.
		if archiveMoved {
			state = StateExtracted
		} else {
			state = StateReused
		}

	case previous.Dirty && archiveMoved:
		switch opts.Resolve {
		case ResolveKeepWorkingCopy:
			state = StateRecovered
		case ResolveTakeArchive:
			state = StateExtracted
		default:
			return nil, StateConflict, ErrConflict
		}

	case previous.Dirty && !previous.Autosave:
		// The mode promised this. See the package comment.
		state = StateDiscarded

	default:
		state = StateRecovered
	}

	if state == StateExtracted || state == StateDiscarded {
		if err := extract(abs, dir); err != nil {
			return nil, state, err
		}
	}

	tree.state = session{
		Version:         sessionVersion,
		Archive:         abs,
		ArchiveSize:     info.Size(),
		ArchiveModified: info.ModTime(),
		Autosave:        opts.Autosave,
		// A recovered tree is still ahead of the archive until it is packed.
		Dirty:    state == StateRecovered,
		OpenedAt: time.Now().UTC(),
	}
	if hasPrevious && state == StateRecovered {
		tree.state.RepackedAt = previous.RepackedAt
	}
	if err := tree.writeSession(); err != nil {
		return nil, state, err
	}
	return tree, state, nil
}

// Dir is the working directory holding this book.
func (t *Tree) Dir() string { return t.dir }

// ContentDir is the directory the archive's members are laid out in.
func (t *Tree) ContentDir() string { return filepath.Join(t.dir, contentDir) }

// Archive is the .draftline this working copy belongs to.
func (t *Tree) Archive() string { return t.archive }

// Dirty reports whether the working copy holds anything the archive does not.
func (t *Tree) Dirty() bool { return t.state.Dirty }

// memberPath resolves an archive member name to a path inside the content
// directory, refusing anything that would escape it.
func (t *Tree) memberPath(name string) (string, error) {
	clean := path.Clean("/" + strings.ReplaceAll(name, `\`, "/"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "", fmt.Errorf("worktree: %q is not a member name", name)
	}
	return filepath.Join(t.ContentDir(), filepath.FromSlash(clean)), nil
}

// ReadMember returns one archive member.
func (t *Tree) ReadMember(name string) ([]byte, error) {
	p, err := t.memberPath(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

// WriteMember replaces one archive member, immediately. This is the write that
// happens the moment a writer changes something: it is one small file, and it
// is what makes the repack a rhythm rather than a cost per keystroke.
func (t *Tree) WriteMember(name string, data []byte) error {
	p, err := t.memberPath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	// Atomic even here. A working copy torn in half by a crash mid-write is a
	// chapter with no end, and it would be recovered as if it were whole.
	if err := fsutil.WriteFileAtomic(p, data, 0o644); err != nil {
		return fmt.Errorf("worktree: write %q: %w", name, err)
	}
	return t.markDirty()
}

// RemoveMember deletes one archive member. Removing something that is not
// there is not an error: a save that drops a chapter should not fail because
// the chapter was already gone.
func (t *Tree) RemoveMember(name string) error {
	p, err := t.memberPath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("worktree: remove %q: %w", name, err)
	}
	return t.markDirty()
}

// Members lists every archive member in the working copy, in archive order.
func (t *Tree) Members() ([]string, error) {
	root := t.ContentDir()
	var names []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("worktree: list members: %w", err)
	}
	sort.Strings(names)
	return names, nil
}

func (t *Tree) markDirty() error {
	if t.state.Dirty {
		return nil
	}
	t.state.Dirty = true
	return t.writeSession()
}

// Repack rebuilds the .draftline from the working copy and puts it in place
// atomically. It is a no-op when nothing has changed, so it is safe to call on
// a timer, on backgrounding, and on close.
func (t *Tree) Repack() error {
	if !t.state.Dirty {
		return nil
	}
	names, err := t.Members()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("worktree: refusing to write an empty book")
	}
	// A book with no manifest is not a book, and writing one over a good
	// archive would destroy it. This is the last gate before the rename.
	if !contains(names, "manifest.json") {
		return errors.New("worktree: refusing to write a book with no manifest.json")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		data, err := t.ReadMember(name)
		if err != nil {
			return fmt.Errorf("worktree: read %q: %w", name, err)
		}
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("worktree: add %q: %w", name, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("worktree: write %q: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("worktree: finalize archive: %w", err)
	}

	// Read the archive back before it replaces a real book. A corrupt writer
	// is worth catching here rather than the next time the book is opened.
	if _, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len())); err != nil {
		return fmt.Errorf("worktree: produced archive is invalid: %w", err)
	}

	// The temp file lives beside the archive so the rename is a rename and not
	// a copy across volumes, and the retry loop inside covers a sync client
	// holding the target open for a moment.
	if err := fsutil.WriteFileAtomic(t.archive, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("worktree: replace %q: %w", t.archive, err)
	}

	info, err := os.Stat(t.archive)
	if err != nil {
		return fmt.Errorf("worktree: stat %q: %w", t.archive, err)
	}
	t.state.ArchiveSize = info.Size()
	t.state.ArchiveModified = info.ModTime()
	t.state.Dirty = false
	t.state.RepackedAt = time.Now().UTC()
	return t.writeSession()
}

// Discard throws the working copy away and lays the archive out again. This is
// what a writer gets when they decline to save in manual mode.
func (t *Tree) Discard() error {
	if err := extract(t.archive, t.dir); err != nil {
		return err
	}
	info, err := os.Stat(t.archive)
	if err != nil {
		return fmt.Errorf("worktree: stat %q: %w", t.archive, err)
	}
	t.state.ArchiveSize = info.Size()
	t.state.ArchiveModified = info.ModTime()
	t.state.Dirty = false
	return t.writeSession()
}

// SetAutosave records a change of mode. It is written through immediately
// because the mode decides what happens to unsaved work after a crash, and a
// mode held only in memory is no use to the process that finds the wreckage.
func (t *Tree) SetAutosave(on bool) error {
	if t.state.Autosave == on {
		return nil
	}
	t.state.Autosave = on
	return t.writeSession()
}

// Autosave reports the mode this session is in.
func (t *Tree) Autosave() bool { return t.state.Autosave }

// Remove deletes the whole working copy. The archive is not touched, so this
// is only safe once anything worth keeping has been packed into it.
func (t *Tree) Remove() error {
	if err := os.RemoveAll(t.dir); err != nil {
		return fmt.Errorf("worktree: remove %q: %w", t.dir, err)
	}
	return nil
}

func (t *Tree) writeSession() error {
	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return fmt.Errorf("worktree: encode session: %w", err)
	}
	return fsutil.WriteFileAtomic(filepath.Join(t.dir, sessionFile), data, 0o644)
}

func readSession(dir string) (session, bool) {
	data, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return session{}, false
	}
	var s session
	if err := json.Unmarshal(data, &s); err != nil {
		return session{}, false
	}
	if s.Version != sessionVersion {
		return session{}, false
	}
	return s, true
}

// extract lays an archive out under dir/content, replacing whatever was there.
func extract(archivePath, dir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("worktree: open %q: %w", archivePath, err)
	}
	defer r.Close()

	if err := ziputil.CheckArchive(r.File); err != nil {
		return fmt.Errorf("worktree: refusing to open archive: %w", err)
	}

	content := filepath.Join(dir, contentDir)
	// Build beside the old content and swap, so a failure halfway through
	// leaves the previous working copy intact rather than half of two books.
	staging := filepath.Join(dir, contentDir+".new")
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("worktree: %w", err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rel, err := safeMemberPath(f.Name)
		if err != nil {
			return err
		}
		target := filepath.Join(staging, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("worktree: %w", err)
		}
		data, err := ziputil.ReadEntry(f)
		if err != nil {
			return fmt.Errorf("worktree: %w", err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("worktree: write %q: %w", rel, err)
		}
	}

	old := filepath.Join(dir, contentDir+".old")
	_ = os.RemoveAll(old)
	if _, err := os.Stat(content); err == nil {
		if err := os.Rename(content, old); err != nil {
			return fmt.Errorf("worktree: retire previous working copy: %w", err)
		}
	}
	if err := os.Rename(staging, content); err != nil {
		// Put the previous copy back rather than leaving no content at all.
		_ = os.Rename(old, content)
		return fmt.Errorf("worktree: install working copy: %w", err)
	}
	_ = os.RemoveAll(old)
	return nil
}

// safeMemberPath rejects archive members that would write outside the working
// copy: absolute paths, and anything climbing out with "..".
func safeMemberPath(name string) (string, error) {
	slashed := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(slashed, "/") || strings.Contains(slashed, "../") ||
		strings.HasSuffix(slashed, "/..") || slashed == ".." {
		return "", fmt.Errorf("worktree: refusing archive member %q", name)
	}
	clean := path.Clean(slashed)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("worktree: refusing archive member %q", name)
	}
	if filepath.IsAbs(filepath.FromSlash(clean)) {
		return "", fmt.Errorf("worktree: refusing archive member %q", name)
	}
	return filepath.FromSlash(clean), nil
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
