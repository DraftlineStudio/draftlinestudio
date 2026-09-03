package readaloud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"draftline/internal/fsutil"
)

// Version 4 replaces the noisy quantized vocoder with the clean FP32 Kokoro
// model. The installer removes the one known obsolete model after migration.
const bundleVersion = "4"
const manifestFileName = "manifest.json"

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

type VerifyResult struct {
	Installed      bool     `json:"installed"`
	Verified       bool     `json:"verified"`
	NativeVerified bool     `json:"native_verified"`
	Version        string   `json:"version"`
	InstalledAt    string   `json:"installed_at"`
	Bytes          int64    `json:"bytes"`
	Corrupt        []string `json:"corrupt,omitempty"`
	Missing        []string `json:"missing,omitempty"`
	Error          string   `json:"error,omitempty"`
}

func WriteInstalledManifest(dir string) error {
	manifest := InstalledManifest{
		ModelID: "draftline-kokoro-native", Version: bundleVersion,
		InstalledAt: time.Now().UTC().Format(time.RFC3339), Files: []ManifestEntry{},
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

// Verify hashes the complete native bundle against source-compiled pins. The
// on-disk manifest is provenance metadata, never an authority.
func Verify(dir string) VerifyResult {
	result := VerifyResult{Corrupt: []string{}, Missing: []string{}}
	allPresent := NativeSupported()
	allValid := NativeSupported()
	for _, art := range Manifest() {
		path := filepath.Join(dir, filepath.FromSlash(art.Name))
		info, err := os.Stat(path)
		if err != nil {
			allPresent = false
			allValid = false
			result.Missing = append(result.Missing, art.Name)
			continue
		}
		result.Installed = true
		if info.Size() != art.Bytes {
			allValid = false
			result.Corrupt = append(result.Corrupt, art.Name)
			continue
		}
		sum, err := sha256File(path)
		if err != nil || sum != art.SHA256 {
			allValid = false
			result.Corrupt = append(result.Corrupt, art.Name)
			continue
		}
		result.Bytes += info.Size()
	}
	ready := allPresent && allValid && Check(dir).NativeInstalled
	result.NativeVerified = ready
	result.Verified = ready

	if manifest := readInstalledManifest(dir); manifest != nil {
		result.Version = manifest.Version
		result.InstalledAt = manifest.InstalledAt
		if ready && manifest.Version != bundleVersion {
			if WriteInstalledManifest(dir) == nil {
				result.Version = bundleVersion
			}
		}
	} else if ready {
		if WriteInstalledManifest(dir) == nil {
			result.Version = bundleVersion
			result.InstalledAt = time.Now().UTC().Format(time.RFC3339)
		}
	}
	return result
}
