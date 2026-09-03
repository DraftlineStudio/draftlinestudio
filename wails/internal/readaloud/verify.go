package readaloud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"draftline/internal/fsutil"
)

// bundleVersion stamps the on-disk manifest; bump when the pinned artifact
// set changes so old installs read as needing repair rather than verified.
const bundleVersion = "1"

const manifestFileName = "manifest.json"

// InstalledManifest is written next to the model files after every
// successful install. It documents exactly what this machine holds — the
// verify/repair/uninstall lifecycle works from it plus the compiled pins.
type InstalledManifest struct {
	ModelID     string          `json:"model_id"`
	Version     string          `json:"version"`
	InstalledAt string          `json:"installed_at"`
	Files       []ManifestEntry `json:"files"`
}

type ManifestEntry struct {
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
	Group  string `json:"group"`
}

// VerifyResult reports a full-hash audit of the install.
type VerifyResult struct {
	Installed   bool     `json:"installed"`    // any core file present at all
	Verified    bool     `json:"verified"`     // every core file hash-matches
	GPUVerified bool     `json:"gpu_verified"` // fp32 model present AND hash-matches
	Version     string   `json:"version"`
	InstalledAt string   `json:"installed_at"`
	Bytes       int64    `json:"bytes"` // verified bytes on disk
	Corrupt     []string `json:"corrupt,omitempty"`
	Missing     []string `json:"missing,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// WriteInstalledManifest records the currently hash-verified files. Called
// after every successful group install; also self-heals a missing manifest
// during Verify when the files themselves check out.
func WriteInstalledManifest(dir string) error {
	manifest := InstalledManifest{
		ModelID:     "onnx-community/Kokoro-82M-v1.0-ONNX@" + kokoroRevision[:12],
		Version:     bundleVersion,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Files:       []ManifestEntry{},
	}
	for _, art := range Manifest() {
		if fileVerifies(filepath.Join(dir, filepath.FromSlash(art.Name)), art) {
			manifest.Files = append(manifest.Files, ManifestEntry{
				Name: art.Name, Bytes: art.Bytes, SHA256: art.SHA256, Group: art.Group,
			})
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(dir, manifestFileName), data, 0644)
}

func readInstalledManifest(dir string) *InstalledManifest {
	data, err := os.ReadFile(filepath.Join(dir, manifestFileName))
	if err != nil {
		return nil
	}
	var manifest InstalledManifest
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	return &manifest
}

// Verify re-hashes every file against the compiled pins (the authority —
// an on-disk manifest can never vouch for itself). The manifest supplies
// provenance metadata and is rewritten when it is missing or stale while
// the files themselves verify.
func Verify(dir string) VerifyResult {
	result := VerifyResult{Corrupt: []string{}, Missing: []string{}}
	gpuPresent := false
	gpuOK := true
	for _, art := range Manifest() {
		path := filepath.Join(dir, filepath.FromSlash(art.Name))
		info, statErr := os.Stat(path)
		present := statErr == nil
		if art.Group == GroupGPU {
			if !present {
				gpuOK = false
				continue
			}
			gpuPresent = true
			if info.Size() != art.Bytes {
				gpuOK = false
				result.Corrupt = append(result.Corrupt, art.Name)
				continue
			}
			if sum, err := sha256File(path); err != nil || sum != art.SHA256 {
				gpuOK = false
				result.Corrupt = append(result.Corrupt, art.Name)
				continue
			}
			result.Bytes += info.Size()
			continue
		}
		if !present {
			result.Missing = append(result.Missing, art.Name)
			continue
		}
		result.Installed = true
		if info.Size() != art.Bytes {
			result.Corrupt = append(result.Corrupt, art.Name)
			continue
		}
		sum, err := sha256File(path)
		if err != nil || sum != art.SHA256 {
			result.Corrupt = append(result.Corrupt, art.Name)
			continue
		}
		result.Bytes += info.Size()
	}
	result.Verified = result.Installed && len(result.Missing) == 0 && !hasCoreCorruption(result.Corrupt)
	result.GPUVerified = gpuPresent && gpuOK

	if manifest := readInstalledManifest(dir); manifest != nil {
		result.Version = manifest.Version
		result.InstalledAt = manifest.InstalledAt
		if manifest.Version != bundleVersion && result.Verified {
			// Pins moved since this manifest was written yet the files match
			// the current pins — refresh the record.
			_ = WriteInstalledManifest(dir)
			result.Version = bundleVersion
		}
	} else if result.Verified {
		// Files are good but the record is missing (pre-manifest install):
		// self-heal rather than reporting corruption.
		if err := WriteInstalledManifest(dir); err == nil {
			result.Version = bundleVersion
			result.InstalledAt = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return result
}

func hasCoreCorruption(corrupt []string) bool {
	for _, name := range corrupt {
		for _, art := range GroupManifest(GroupCore) {
			if art.Name == name {
				return true
			}
		}
	}
	return false
}
