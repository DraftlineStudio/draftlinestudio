package main

// Manual update check against the public GitHub releases. Nothing here runs
// in the background: the settings dialog's "Check for Updates" button calls
// CheckForUpdates, and DownloadUpdate only runs when the writer asks for the
// download. Downloads are verified against the release's SHA256SUMS.txt
// before anything is launched.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepoAPI = "https://api.github.com/repos/DraftlineStudio/draftlinestudio/releases"
	// A release asset larger than this is not a Draftline package.
	maxUpdateDownloadBytes = 600 << 20
)

type UpdateCheckResult struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	LatestLabel     string `json:"latest_label,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
	ReleaseNotes    string `json:"release_notes,omitempty"`
	AssetName       string `json:"asset_name,omitempty"`
	AssetSize       int64  `json:"asset_size,omitempty"`
	Error           string `json:"error,omitempty"`
}

type UpdateDownloadResult struct {
	Path     string `json:"path,omitempty"`
	Launched bool   `json:"launched"`
	Error    string `json:"error,omitempty"`
}

type githubReleaseAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string               `json:"tag_name"`
	HTMLURL string               `json:"html_url"`
	Body    string               `json:"body"`
	Draft   bool                 `json:"draft"`
	Assets  []githubReleaseAsset `json:"assets"`
}

// parseBuildVersion turns "0.17.02563" into comparable integers.
func parseBuildVersion(version string) ([3]int, bool) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var result [3]int
	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return [3]int{}, false
		}
		result[index] = value
	}
	return result, true
}

// parseReleaseTag splits a release tag ("v0.17.02563-beta") into its numeric
// version and the full label used in asset file names ("0.17.02563-beta").
func parseReleaseTag(tag string) (version string, label string, ok bool) {
	label = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	version = label
	if dash := strings.IndexByte(label, '-'); dash >= 0 {
		version = label[:dash]
	}
	if _, valid := parseBuildVersion(version); !valid {
		return "", "", false
	}
	return version, label, true
}

func versionNewer(candidate, current string) bool {
	a, okA := parseBuildVersion(candidate)
	b, okB := parseBuildVersion(current)
	if !okA || !okB {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return a[index] > b[index]
		}
	}
	return false
}

// platformAssetSuffix names the release asset for this build's platform,
// matching the packaging workflow's naming exactly.
func platformAssetSuffix(goos, goarch string) string {
	switch goos {
	case "windows":
		if goarch == "amd64" {
			return "-windows-amd64-setup.exe"
		}
	case "darwin":
		return "-macos-universal.dmg"
	case "linux":
		if goarch == "amd64" {
			return "-linux-x86_64.AppImage"
		}
	}
	return ""
}

func pickReleaseAsset(release *githubRelease, goos, goarch string) *githubReleaseAsset {
	suffix := platformAssetSuffix(goos, goarch)
	if suffix == "" {
		return nil
	}
	for index := range release.Assets {
		if strings.HasSuffix(release.Assets[index].Name, suffix) {
			return &release.Assets[index]
		}
	}
	return nil
}

// parseSHA256Sums reads sha256sum output ("<hex>  <name>") into a lookup.
func parseSHA256Sums(content string) map[string]string {
	sums := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || len(fields[0]) != 64 {
			continue
		}
		sums[strings.TrimPrefix(fields[1], "*")] = strings.ToLower(fields[0])
	}
	return sums
}

func updateHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func githubGet(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Draftline/"+AppVersion)
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("GitHub responded with %s", response.Status)
	}
	return response, nil
}

// latestRelease returns the newest published release, preferring the
// releases/latest endpoint and falling back to the release list so
// pre-release builds are still offered.
func latestRelease(ctx context.Context, client *http.Client) (*githubRelease, error) {
	if response, err := githubGet(ctx, client, updateRepoAPI+"/latest"); err == nil {
		defer response.Body.Close()
		release := &githubRelease{}
		if decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(release); decodeErr == nil && release.TagName != "" {
			return release, nil
		}
	}
	response, err := githubGet(ctx, client, updateRepoAPI+"?per_page=10")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	releases := []githubRelease{}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&releases); err != nil {
		return nil, err
	}
	for index := range releases {
		if !releases[index].Draft {
			return &releases[index], nil
		}
	}
	return nil, fmt.Errorf("no published releases")
}

// CheckForUpdates compares the running version against the newest published
// GitHub release. It never downloads anything.
func (a *App) CheckForUpdates() UpdateCheckResult {
	result := UpdateCheckResult{CurrentVersion: AppVersion}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	release, err := latestRelease(ctx, updateHTTPClient(30*time.Second))
	if err != nil {
		result.Error = "Could not check for updates: " + err.Error()
		return result
	}
	version, label, ok := parseReleaseTag(release.TagName)
	if !ok {
		result.Error = fmt.Sprintf("The newest release tag %q is not a version Draftline understands.", release.TagName)
		return result
	}
	result.LatestVersion = version
	result.LatestLabel = label
	result.ReleaseURL = release.HTMLURL
	result.ReleaseNotes = release.Body
	if len(result.ReleaseNotes) > 4000 {
		result.ReleaseNotes = result.ReleaseNotes[:4000] + "…"
	}
	result.UpdateAvailable = versionNewer(version, AppVersion)
	if asset := pickReleaseAsset(release, runtime.GOOS, runtime.GOARCH); asset != nil {
		result.AssetName = asset.Name
		result.AssetSize = asset.Size
	}
	return result
}

// DownloadUpdate downloads this platform's asset from the newest release,
// verifies it against the release's SHA256SUMS.txt, and opens it (the NSIS
// installer on Windows, the mounted disk image on macOS, the containing
// folder on Linux). The user drives every step from there.
func (a *App) DownloadUpdate() UpdateDownloadResult {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	client := updateHTTPClient(15 * time.Minute)

	release, err := latestRelease(ctx, client)
	if err != nil {
		return UpdateDownloadResult{Error: "Could not reach GitHub: " + err.Error()}
	}
	version, _, ok := parseReleaseTag(release.TagName)
	if !ok || !versionNewer(version, AppVersion) {
		return UpdateDownloadResult{Error: "No newer release is available."}
	}
	asset := pickReleaseAsset(release, runtime.GOOS, runtime.GOARCH)
	if asset == nil {
		return UpdateDownloadResult{Error: "The newest release has no package for this platform yet."}
	}
	if asset.Size > maxUpdateDownloadBytes {
		return UpdateDownloadResult{Error: "The release asset is unexpectedly large; refusing to download."}
	}

	expected := ""
	for index := range release.Assets {
		if release.Assets[index].Name == "SHA256SUMS.txt" {
			sumsResponse, sumsErr := githubGet(ctx, client, release.Assets[index].DownloadURL)
			if sumsErr != nil {
				return UpdateDownloadResult{Error: "Could not fetch the release checksums: " + sumsErr.Error()}
			}
			content, readErr := io.ReadAll(io.LimitReader(sumsResponse.Body, 1<<20))
			sumsResponse.Body.Close()
			if readErr != nil {
				return UpdateDownloadResult{Error: "Could not read the release checksums: " + readErr.Error()}
			}
			expected = parseSHA256Sums(string(content))[asset.Name]
		}
	}
	if expected == "" {
		return UpdateDownloadResult{Error: "The release does not include a checksum for this package; refusing to download."}
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	targetDir := filepath.Join(cacheDir, "Draftline", "updates")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return UpdateDownloadResult{Error: "Could not create the download folder: " + err.Error()}
	}
	targetPath := filepath.Join(targetDir, asset.Name)

	response, err := githubGet(ctx, client, asset.DownloadURL)
	if err != nil {
		return UpdateDownloadResult{Error: "Download failed: " + err.Error()}
	}
	defer response.Body.Close()

	file, err := os.Create(targetPath)
	if err != nil {
		return UpdateDownloadResult{Error: "Could not write the download: " + err.Error()}
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(response.Body, maxUpdateDownloadBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(targetPath)
		return UpdateDownloadResult{Error: "Download failed before completing."}
	}
	if actual := hex.EncodeToString(hasher.Sum(nil)); actual != expected {
		os.Remove(targetPath)
		return UpdateDownloadResult{Error: "The downloaded file did not match the release checksum and was discarded."}
	}

	launched := openDownloadedUpdate(targetPath)
	return UpdateDownloadResult{Path: targetPath, Launched: launched}
}

func openDownloadedUpdate(path string) bool {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("cmd", "/C", "start", "", path)
	case "darwin":
		command = exec.Command("open", path)
	default:
		command = exec.Command("xdg-open", filepath.Dir(path))
	}
	return command.Start() == nil
}
