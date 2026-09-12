package instancelock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// point the lock dir at a temp location for the test's duration.
func tempLockDir(t *testing.T) {
	t.Helper()
	t.Setenv(configDirEnv(), t.TempDir())
}

// configDirEnv names the env var os.UserConfigDir reads per platform.
func configDirEnv() string {
	if os.PathSeparator == '\\' {
		return "AppData"
	}
	return "XDG_CONFIG_HOME"
}

func TestAcquireReleaseRoundTrip(t *testing.T) {
	tempLockDir(t)
	book := filepath.Join(t.TempDir(), "novel.draftline")

	lock, owner, err := Acquire(book)
	if err != nil || owner != nil || lock == nil {
		t.Fatalf("first acquire: lock=%v owner=%v err=%v", lock, owner, err)
	}
	// Same process re-acquiring the same book must succeed, not self-collide.
	again, owner2, err := Acquire(book)
	if err != nil || owner2 != nil || again == nil {
		t.Fatalf("re-acquire by same pid: owner=%v err=%v", owner2, err)
	}
	lock.Release()
	if CurrentOwner(book) != nil {
		t.Fatal("owner should be gone after release")
	}
}

func TestLivingForeignOwnerIsReported(t *testing.T) {
	tempLockDir(t)
	book := filepath.Join(t.TempDir(), "novel.draftline")
	// The test process's parent is a real, living, foreign PID.
	writeLock(t, book, os.Getppid())

	lock, owner, err := Acquire(book)
	if err != nil {
		t.Fatalf("acquire err: %v", err)
	}
	if lock != nil || owner == nil || owner.PID != os.Getppid() {
		t.Fatalf("expected living owner %d, got lock=%v owner=%v", os.Getppid(), lock, owner)
	}
	if co := CurrentOwner(book); co == nil || co.PID != os.Getppid() {
		t.Fatalf("CurrentOwner should report the living owner, got %v", co)
	}
}

func TestStaleLockIsStolen(t *testing.T) {
	tempLockDir(t)
	book := filepath.Join(t.TempDir(), "novel.draftline")
	// A PID that cannot be alive: beyond typical ranges and long dead.
	writeLock(t, book, 999999999)

	lock, owner, err := Acquire(book)
	if err != nil || owner != nil || lock == nil {
		t.Fatalf("stale lock should be stolen: lock=%v owner=%v err=%v", lock, owner, err)
	}
	lock.Release()
}

func TestCorruptLockIsCleared(t *testing.T) {
	tempLockDir(t)
	book := filepath.Join(t.TempDir(), "novel.draftline")
	file, err := lockFileFor(book)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, owner, err := Acquire(book)
	if err != nil || owner != nil || lock == nil {
		t.Fatalf("corrupt lock should be cleared: lock=%v owner=%v err=%v", lock, owner, err)
	}
	lock.Release()
}

func TestNormalizeMapsEquivalentPathsToOneLock(t *testing.T) {
	tempLockDir(t)
	dir := t.TempDir()
	book := filepath.Join(dir, "novel.draftline")
	alias := filepath.Join(dir, ".", "novel.draftline")

	lock, _, err := Acquire(book)
	if err != nil || lock == nil {
		t.Fatalf("acquire: %v", err)
	}
	defer lock.Release()
	// The alias resolves to the same lock, held by us → re-acquire, no owner.
	other, owner, err := Acquire(alias)
	if err != nil || owner != nil || other == nil {
		t.Fatalf("alias path should map to the same held lock: owner=%v err=%v", owner, err)
	}
}

func writeLock(t *testing.T, book string, pid int) {
	t.Helper()
	file, err := lockFileFor(book)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(Owner{PID: pid, BookPath: book, Acquired: time.Now().Format(time.RFC3339)})
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
