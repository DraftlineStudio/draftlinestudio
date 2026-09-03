package readaloud

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Progress reports byte-level download state for one manifest file plus the
// bundle as a whole. Emitted at most a few times per second.
type Progress struct {
	File            string `json:"file"`
	FileIndex       int    `json:"file_index"`
	FileCount       int    `json:"file_count"`
	Received        int64  `json:"received"`
	Total           int64  `json:"total"`
	OverallReceived int64  `json:"overall_received"`
	OverallTotal    int64  `json:"overall_total"`
}

// progressInterval throttles onProgress callbacks during a streaming download.
const progressInterval = 250 * time.Millisecond

var httpClient = &http.Client{Timeout: 30 * time.Minute}

// Install downloads every missing core-bundle artifact into dir, verifying
// each against its pinned size and SHA-256 before renaming it into place.
// Files that already verify are skipped, so a cancelled install resumes at
// file granularity. On cancellation or error the in-flight .partial file is
// removed; completed files are kept.
func Install(ctx context.Context, dir string, onProgress func(Progress)) error {
	return installManifest(ctx, dir, GroupManifest(GroupCore), onProgress)
}

// InstallGroup downloads one artifact group (e.g. the optional GPU model)
// with the same verification and resume semantics as Install.
func InstallGroup(ctx context.Context, dir, group string, onProgress func(Progress)) error {
	return installManifest(ctx, dir, GroupManifest(group), onProgress)
}

// InstallNative installs the CPU-optimized native model and support bundle.
// Each pass is resumable and checksum verified. Unsupported
// platforms fail explicitly instead of downloading unusable binaries.
func InstallNative(ctx context.Context, dir string, onProgress func(Progress)) error {
	if !NativeSupported() {
		return fmt.Errorf("native read aloud is not available for this platform")
	}
	manifest := GroupManifest(GroupNative)
	if err := installManifest(ctx, dir, manifest, onProgress); err != nil {
		return err
	}
	return extractEspeakData(ctx, dir)
}

func extractEspeakData(ctx context.Context, dir string) error {
	modelDir := filepath.Join(dir, "native", "model")
	destination := filepath.Join(modelDir, "espeak-ng-data")
	if info, err := os.Stat(filepath.Join(destination, "phontab")); err == nil && !info.IsDir() {
		return nil
	}
	staging := filepath.Join(modelDir, ".espeak-ng-data.extracting")
	if filepath.Base(destination) != "espeak-ng-data" || filepath.Base(staging) != ".espeak-ng-data.extracting" {
		return fmt.Errorf("refusing unsafe espeak extraction path")
	}
	_ = os.RemoveAll(staging)
	if err := os.MkdirAll(staging, 0755); err != nil {
		return err
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(staging)
		}
	}()

	archive, err := os.Open(filepath.Join(modelDir, "espeak-ng-data.tar.bz2"))
	if err != nil {
		return err
	}
	defer archive.Close()
	reader := tar.NewReader(bzip2.NewReader(archive))
	var total int64
	files := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read espeak archive: %w", err)
		}
		rel, ok := strings.CutPrefix(filepath.ToSlash(header.Name), "espeak-ng-data/")
		if !ok || rel == "" {
			continue
		}
		target, err := secureJoin(staging, rel)
		if err != nil {
			return fmt.Errorf("unsafe espeak archive entry %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			files++
			total += header.Size
			if files > 4096 || total > 64*1024*1024 {
				return fmt.Errorf("espeak archive exceeds safety limits")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(out, reader, header.Size)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported espeak archive entry %q", header.Name)
		}
	}
	if _, err := os.Stat(filepath.Join(staging, "phontab")); err != nil {
		return fmt.Errorf("espeak archive is missing phontab")
	}
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	if err := os.Rename(staging, destination); err != nil {
		return err
	}
	failed = false
	return nil
}

func installManifest(ctx context.Context, dir string, manifest []Artifact, onProgress func(Progress)) error {
	var overallTotal int64
	for _, a := range manifest {
		overallTotal += a.Bytes
	}
	var overallDone int64

	for i, art := range manifest {
		dest := filepath.Join(dir, filepath.FromSlash(art.Name))
		if fileVerifies(dest, art) {
			overallDone += art.Bytes
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := downloadArtifact(ctx, dest, art, func(received int64) {
			if onProgress != nil {
				onProgress(Progress{
					File:            art.Name,
					FileIndex:       i,
					FileCount:       len(manifest),
					Received:        received,
					Total:           art.Bytes,
					OverallReceived: overallDone + received,
					OverallTotal:    overallTotal,
				})
			}
		}); err != nil {
			return fmt.Errorf("%s: %w", art.Name, err)
		}
		overallDone += art.Bytes
	}
	if onProgress != nil {
		onProgress(Progress{
			FileIndex: len(manifest), FileCount: len(manifest),
			OverallReceived: overallDone, OverallTotal: overallTotal,
		})
	}
	// Record what this machine now holds; verify/repair/uninstall work from
	// this manifest plus the compiled pins.
	return WriteInstalledManifest(dir)
}

// fileVerifies reports whether dest already matches the pinned artifact.
func fileVerifies(dest string, art Artifact) bool {
	info, err := os.Stat(dest)
	if err != nil || info.Size() != art.Bytes {
		return false
	}
	sum, err := sha256File(dest)
	return err == nil && sum == art.SHA256
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadArtifact(ctx context.Context, dest string, art Artifact, onReceived func(int64)) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, art.URL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}

	partial := dest + ".partial"
	out, err := os.OpenFile(partial, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cleanup := func() {
		out.Close()
		_ = os.Remove(partial)
	}

	hasher := sha256.New()
	// Refuse anything beyond the pinned size: a longer body can never verify.
	limited := io.LimitReader(resp.Body, art.Bytes+1)
	buf := make([]byte, 256*1024)
	var received int64
	lastReport := time.Time{}
	for {
		n, readErr := limited.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				cleanup()
				return err
			}
			hasher.Write(buf[:n])
			received += int64(n)
			if received > art.Bytes {
				cleanup()
				return fmt.Errorf("body exceeds pinned size %d — refusing", art.Bytes)
			}
			if onReceived != nil && time.Since(lastReport) >= progressInterval {
				lastReport = time.Now()
				onReceived(received)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			cleanup()
			return readErr
		}
	}
	if received != art.Bytes {
		cleanup()
		return fmt.Errorf("got %d bytes, pinned size is %d", received, art.Bytes)
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); sum != art.SHA256 {
		cleanup()
		return fmt.Errorf("checksum mismatch — refusing to install")
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(partial)
		return err
	}
	if onReceived != nil {
		onReceived(received)
	}
	return os.Rename(partial, dest)
}
