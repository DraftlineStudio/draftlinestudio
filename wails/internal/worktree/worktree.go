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
//	<root>/.The Weather House.draftline-4c1e7a90b332/
//	    content/       the archive, extracted member for member
//	    session.json   what this working copy is and how it stands
//
// The leading dot hides the directory on macOS and Linux; on Windows the
// application data directory is out of the way already.
//
// The suffix is the book's own identifier (types.Metadata.BookID), NOT a hash
// of its path. Keying on the path is the obvious thing and it is wrong: drag a
// project into a Dropbox folder and its path changes, so the working copy
// holding its unsaved chapters is stranded under a name nothing will look for
// again. An identifier survives the move, the rename, and the same book being
// opened on a second device. Only the suffix is load-bearing: a renamed book
// is found by it and the readable half is brought back into line, so the
// directory listing stays honest without the name ever being the key.
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
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"draftline/internal/fsutil"
	"draftline/internal/types"
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
	BookID  string `json:"book_id"`
	// Archive is where the book was last seen. It is informational: the
	// working copy is found by identifier, and this is what lets a caller say
	// which book an orphaned working copy belonged to.
	Archive string `json:"archive"`
	// ArchiveHash is the archive as it stood when this working copy last
	// agreed with it. It is a content hash rather than a size and a timestamp
	// because a book that moves between folders keeps its content and loses
	// its timestamp, and confusing "moved" with "changed" either re-extracts
	// over unsaved work or writes a stale copy over a newer one.
	ArchiveHash string `json:"archive_hash"`
	// Autosave records the mode the session was opened in, because what to do
	// with an unsaved working copy depends entirely on it.
	Autosave   bool      `json:"autosave"`
	Dirty      bool      `json:"dirty"`
	OpenedAt   time.Time `json:"opened_at"`
	RepackedAt time.Time `json:"repacked_at,omitempty"`
}

// hashArchive fingerprints a .draftline. Opening a book is not a hot path and
// a manuscript is a few hundred kilobytes, so this costs milliseconds and buys
// an unambiguous answer to "is this the same file I left".
func hashArchive(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("worktree: read %q: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("worktree: read %q: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Tree is one book, open as a directory.
type Tree struct {
	dir     string
	archive string
	bookID  string
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

// DirFor reports the working directory for a book, without touching disk.
//
// The key is the book's identifier, never its path. Keying on the path looked
// obvious and was wrong: dragging a project into a Dropbox folder would strand
// the working copy holding its unsaved chapters under the old path's name,
// where nothing would ever look for it again. An identifier survives the move,
// the rename, and being opened on a second device.
//
// The filename is only there so a human can read the directory listing, and it
// is allowed to go stale when a book is renamed.
func DirFor(root, archivePath, bookID string) string {
	name := filepath.Base(archivePath)
	return filepath.Join(root, fmt.Sprintf(".%s-%s", name, types.ShortBookID(bookID)))
}

// locateDir finds this book's existing working copy whatever the book is
// called now, and brings its name back into line if the book was renamed.
//
// The identifier is the key and the filename is decoration, but the filename
// is part of the directory name, so a rename has to be followed or the working
// copy is lost exactly the way keying on the path lost it.
func locateDir(root, archivePath, bookID string) string {
	want := DirFor(root, archivePath, bookID)
	if _, err := os.Stat(filepath.Join(want, sessionFile)); err == nil {
		return want
	}

	suffix := "-" + types.ShortBookID(bookID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return want
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		found := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(found, sessionFile)); err != nil {
			continue
		}
		// Follow the rename. If the move fails — something holding the
		// directory open — the working copy is still usable where it is,
		// which matters more than its name being tidy.
		if err := os.Rename(found, want); err != nil {
			return found
		}
		return want
	}
	return want
}

// IdentifyArchive reads a book's identifier out of its manifest without
// unpacking the manuscript, deriving one the way every other reader derives it
// if the project predates the field.
func IdentifyArchive(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("worktree: open %q: %w", archivePath, err)
	}
	defer r.Close()

	data, err := ziputil.ReadNamed(r.File, "manifest.json", false)
	if err != nil {
		return "", fmt.Errorf("worktree: %q has no manifest: %w", archivePath, err)
	}
	var manifest struct {
		Metadata types.Metadata `json:"metadata"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("worktree: manifest of %q could not be read: %w", archivePath, err)
	}
	manifest.Metadata.EnsureBookID()
	return manifest.Metadata.BookID, nil
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
	if _, err := os.Stat(abs); err != nil {
		return nil, StateExtracted, fmt.Errorf("worktree: open %q: %w", archivePath, err)
	}

	bookID, err := IdentifyArchive(abs)
	if err != nil {
		return nil, StateExtracted, err
	}
	hash, err := hashArchive(abs)
	if err != nil {
		return nil, StateExtracted, err
	}

	dir := locateDir(opts.Root, abs, bookID)
	tree := &Tree{dir: dir, archive: abs, bookID: bookID}

	previous, hasPrevious := readSession(dir)
	// A book that has simply moved — dragged into a synced folder, renamed —
	// is the same book with the same contents, and its working copy comes with
	// it. Only the contents changing matters, which is what the hash answers.
	archiveChanged := hasPrevious && previous.ArchiveHash != hash

	state := StateExtracted
	switch {
	case !hasPrevious:
		state = StateExtracted

	case !previous.Dirty:
		// Nothing unsaved. Reuse the tree when the archive is untouched,
		// re-extract when it is not: an unchanged tree is worth nothing and
		// the archive is always right in this branch.
		if archiveChanged {
			state = StateExtracted
		} else {
			state = StateReused
		}

	case previous.Dirty && archiveChanged:
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
		Version:     sessionVersion,
		BookID:      bookID,
		Archive:     abs,
		ArchiveHash: hash,
		Autosave:    opts.Autosave,
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

// BookID is the identifier this working copy is filed under.
func (t *Tree) BookID() string { return t.bookID }

// Orphan is a working copy whose book is no longer where it was last seen.
type Orphan struct {
	Dir string
	// BookID and Archive say which book it was, so a caller can offer to
	// restore it somewhere rather than describing a hex string to a writer.
	BookID  string
	Archive string
	// Dirty working copies hold writing that never reached a .draftline.
	// Those are the ones worth telling somebody about.
	Dirty    bool
	OpenedAt time.Time
}

// Orphans lists working copies whose archive is no longer at its last known
// path. Two things make one: a book deleted, and a book moved by something
// other than Draftline while it was closed — in which case the next open finds
// it again by identifier, and the orphan is this one's stale twin.
//
// Nothing is deleted here. A caller decides, because a dirty orphan is
// somebody's unsaved afternoon.
func Orphans(root string) ([]Orphan, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("worktree: list %q: %w", root, err)
	}
	var out []Orphan
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		state, ok := readSession(dir)
		if !ok {
			continue
		}
		if _, err := os.Stat(state.Archive); err == nil {
			continue
		}
		out = append(out, Orphan{
			Dir:      dir,
			BookID:   state.BookID,
			Archive:  state.Archive,
			Dirty:    state.Dirty,
			OpenedAt: state.OpenedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Dir < out[j].Dir })
	return out, nil
}

// Prune removes orphaned working copies that hold nothing unsaved, and reports
// the ones it left behind because they do.
func Prune(root string) ([]Orphan, error) {
	orphans, err := Orphans(root)
	if err != nil {
		return nil, err
	}
	var kept []Orphan
	for _, orphan := range orphans {
		if orphan.Dirty {
			kept = append(kept, orphan)
			continue
		}
		if err := os.RemoveAll(orphan.Dir); err != nil {
			return kept, fmt.Errorf("worktree: remove %q: %w", orphan.Dir, err)
		}
	}
	return kept, nil
}

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

	hash, err := hashArchive(t.archive)
	if err != nil {
		return err
	}
	t.state.ArchiveHash = hash
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
	hash, err := hashArchive(t.archive)
	if err != nil {
		return err
	}
	t.state.ArchiveHash = hash
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
