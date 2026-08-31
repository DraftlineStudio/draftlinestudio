package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"strings"
	"time"

	goruntime "runtime"

	"draftline/internal/platform"
	"draftline/internal/types"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Pinned LTS version. Can be bumped in future releases.
const nodeVersion = "v22.14.0"

// nodeSHA256 pins the expected checksum of each Node.js artifact. When bumping
// nodeVersion, refresh every entry from
// https://nodejs.org/dist/<version>/SHASUMS256.txt. Platforms not listed here
// are refused rather than installed unverified.
var nodeSHA256 = map[string]string{
	"node-v22.14.0-win-x64.zip":         "55b639295920b219bb2acbcfa00f90393a2789095b7323f79475c9f34795f217",
	"node-v22.14.0-win-arm64.zip":       "2d71f5f9b2fffa33baa108c07d74b0d24e0c3dd8f441d567772ae0e3dd4b1a22",
	"node-v22.14.0-darwin-x64.tar.gz":   "6698587713ab565a94a360e091df9f6d91c8fadda6d00f0cf6526e9b40bed250",
	"node-v22.14.0-darwin-arm64.tar.gz": "e9404633bc02a5162c5c573b1e2490f5fb44648345d64a958b17e325729a5e42",
	"node-v22.14.0-linux-x64.tar.gz":    "9d942932535988091034dc94cc5f42b6dc8784d6366df3a36c4c9ccb3996f0c2",
	"node-v22.14.0-linux-arm64.tar.gz":  "8cf30ff7250f9463b53c18f89c6c606dfda70378215b2c905d0a9a8b08bd45e0",
}

// Pinned Claude Code CLI version; updates ride app releases so the supply
// chain stays auditable.
const claudeCodeVersion = "2.1.251"

// Pinned OpenAI Codex CLI version (ChatGPT-account AI mode); same policy.
const codexCLIVersion = "0.151.0"

// Extraction limits for the Node distribution (npm ships thousands of small
// files; node's binary is ~90 MB uncompressed).
const (
	maxNodeDownload   = 80 << 20
	maxExtractEntry   = 200 << 20
	maxExtractTotal   = 500 << 20
	maxExtractEntries = 50_000
)

// ── Path helpers ────────────────────────────────────────────────────────────

func draftlineAIToolsDir() string {
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		dir, _ = os.UserConfigDir()
	}
	p := filepath.Join(dir, "draftline", "ai")
	_ = os.MkdirAll(p, 0700)
	_ = os.Chmod(p, 0700)
	return p
}

// nodeInstallDir is where Draftline extracts a portable Node.js distribution.
func nodeInstallDir() string {
	return filepath.Join(draftlineAIToolsDir(), "node")
}

func claudeInstallDir() string {
	return filepath.Join(draftlineAIToolsDir(), "claude")
}

func codexInstallDir() string {
	return filepath.Join(draftlineAIToolsDir(), "codex")
}

// nodeInstallBinDir is the directory inside our portable Node.js that contains executables.
func nodeInstallBinDir() string {
	d := nodeInstallDir()
	if goruntime.GOOS == "windows" {
		return d // node.exe, npm.cmd are at the root on Windows zips
	}
	return filepath.Join(d, "bin")
}

func bundledNodeBin() string {
	d := nodeInstallDir()
	if goruntime.GOOS == "windows" {
		return filepath.Join(d, "node.exe")
	}
	return filepath.Join(d, "bin", "node")
}

func bundledNpmBin() string {
	d := nodeInstallDir()
	if goruntime.GOOS == "windows" {
		return filepath.Join(d, "npm.cmd")
	}
	return filepath.Join(d, "bin", "npm")
}

// bundledClaudeBin returns the expected path to the claude binary in our npm global dir.
func bundledClaudeBin() string {
	d := claudeInstallDir()
	if goruntime.GOOS == "windows" {
		return filepath.Join(d, "claude.cmd")
	}
	return filepath.Join(d, "bin", "claude")
}

// bundledCodexBin returns the expected path to the codex binary in our npm global dir.
func bundledCodexBin() string {
	d := codexInstallDir()
	if goruntime.GOOS == "windows" {
		return filepath.Join(d, "codex.cmd")
	}
	return filepath.Join(d, "bin", "codex")
}

// resolveCodexBin returns only Draftline's managed, version-pinned Codex CLI.
// PATH is deliberately ignored so an unrelated global update cannot silently
// change the security flags or behavior Draftline relies on.
func resolveCodexBin() string {
	if managedCodexIsPinned() {
		return bundledCodexBin()
	}
	return ""
}

// resolveNpmBin is used only to install managed tools. Prefer Draftline's
// checksum-pinned runtime, with system npm as a bootstrap fallback.
func resolveNpmBin() string {
	p := bundledNpmBin()
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	return ""
}

func installedNpmPackageVersion(prefix string, packagePath ...string) string {
	parts := []string{prefix}
	if goruntime.GOOS != "windows" {
		parts = append(parts, "lib")
	}
	parts = append(parts, "node_modules")
	parts = append(parts, packagePath...)
	data, err := os.ReadFile(filepath.Join(append(parts, "package.json")...))
	if err != nil {
		return ""
	}
	var manifest struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return ""
	}
	return strings.TrimSpace(manifest.Version)
}

func managedClaudeIsPinned() bool {
	if installedNpmPackageVersion(claudeInstallDir(), "@anthropic-ai", "claude-code") != claudeCodeVersion {
		return false
	}
	_, err := os.Stat(bundledClaudeBin())
	return err == nil
}

func managedCodexIsPinned() bool {
	if installedNpmPackageVersion(codexInstallDir(), "@openai", "codex") != codexCLIVersion {
		return false
	}
	_, err := os.Stat(bundledCodexBin())
	return err == nil
}

// cmdExec wraps exec.CommandContext so that on Windows, .cmd and .bat files
// are run through cmd.exe /S /C with a hand-built command line — both plain
// exec and the naive /C form split the script path at spaces once other quoted
// arguments are present.
func cmdExec(ctx context.Context, path string, args ...string) *exec.Cmd {
	if goruntime.GOOS == "windows" {
		lp := strings.ToLower(path)
		if strings.HasSuffix(lp, ".cmd") || strings.HasSuffix(lp, ".bat") {
			return platform.BatchCommand(ctx, path, args...)
		}
	}
	return exec.CommandContext(ctx, path, args...)
}

// claudeExec builds an exec.Cmd to run Claude Code. On Windows it bypasses the
// claude.cmd shim entirely: claude-code 2.x ships a native bin/claude.exe, and
// 1.x shipped cli.js for node — either avoids cmd.exe quoting quirks.
// On macOS/Linux the shell script is directly executable so cmdExec is used as-is.
func claudeExec(ctx context.Context, claudePath string, args ...string) *exec.Cmd {
	if goruntime.GOOS != "windows" {
		return cmdExec(ctx, claudePath, args...)
	}

	// claude.cmd lives in the npm global prefix dir, so dirname(claude.cmd) is
	// the prefix and the package sits under its node_modules.
	pkgDir := filepath.Join(filepath.Dir(claudePath), "node_modules", "@anthropic-ai", "claude-code")

	// claude-code 2.x: native executable, run it directly.
	exe := filepath.Join(pkgDir, "bin", "claude.exe")
	if _, err := os.Stat(exe); err == nil {
		return exec.CommandContext(ctx, exe, args...)
	}

	// claude-code 1.x: cli.js run via node.
	cliJS := filepath.Join(pkgDir, "cli.js")
	if _, err := os.Stat(cliJS); err != nil {
		// neither layout found — fall back to running the shim via cmd.exe
		return cmdExec(ctx, claudePath, args...)
	}

	// Prefer our bundled node; fall back to whatever node is on the system PATH.
	nodeBin := bundledNodeBin()
	if _, err := os.Stat(nodeBin); err != nil {
		p, err := exec.LookPath("node")
		if err != nil {
			return cmdExec(ctx, claudePath, args...) // last resort fallback
		}
		nodeBin = p
	}

	return exec.CommandContext(ctx, nodeBin, append([]string{cliJS}, args...)...)
}

// codexExec builds an exec.Cmd to run the Codex CLI. On Windows it bypasses
// the codex.cmd shim when possible: @openai/codex ships a native binary under
// the package's vendor/ tree, and older layouts a bin/codex.js for node.
// Falling back to the shim via cmdExec is safe — every codex invocation passes
// only constant args, with the prompt piped via stdin.
func codexExec(ctx context.Context, codexPath string, args ...string) *exec.Cmd {
	if goruntime.GOOS != "windows" {
		return cmdExec(ctx, codexPath, args...)
	}

	pkgDir := filepath.Join(filepath.Dir(codexPath), "node_modules", "@openai", "codex")

	// Native binary layouts (target triple varies by architecture).
	targets := []string{"x86_64-pc-windows-msvc", "aarch64-pc-windows-msvc"}
	for _, t := range targets {
		for _, rel := range []string{
			filepath.Join("vendor", t, "codex", "codex.exe"),
			filepath.Join("bin", "codex-"+t+".exe"),
		} {
			exe := filepath.Join(pkgDir, rel)
			if _, err := os.Stat(exe); err == nil {
				return exec.CommandContext(ctx, exe, args...)
			}
		}
	}

	// JS launcher via node.
	cliJS := filepath.Join(pkgDir, "bin", "codex.js")
	if _, err := os.Stat(cliJS); err == nil {
		nodeBin := bundledNodeBin()
		if _, err := os.Stat(nodeBin); err != nil {
			if p, err := exec.LookPath("node"); err == nil {
				nodeBin = p
			} else {
				return cmdExec(ctx, codexPath, args...)
			}
		}
		return exec.CommandContext(ctx, nodeBin, append([]string{cliJS}, args...)...)
	}

	return cmdExec(ctx, codexPath, args...)
}

// resolveClaudeBin returns only Draftline's managed, version-pinned Claude CLI.
func resolveClaudeBin() string {
	if managedClaudeIsPinned() {
		return bundledClaudeBin()
	}
	return ""
}

// ── Setup orchestration ─────────────────────────────────────────────────────

// SetupClaudeCode orchestrates the full Claude Code setup in one call:
//  1. Download and verify Draftline's pinned portable Node.js runtime
//  2. Install the exact Claude Code version into its private AI directory
//
// Progress is streamed to the frontend via "setup:progress" events.
// The function blocks until setup is complete (or fails).
func (a *App) SetupClaudeCode() types.ClaudeCodeStatus {
	emit := func(msg string) {
		runtime.EventsEmit(a.ctx, "setup:progress", msg)
	}

	// ── Step 1: Ensure npm is available ─────────────────────────────────────
	npmPath := bundledNpmBin()
	if _, err := os.Stat(npmPath); err != nil {
		emit("step:node")
		emit("Downloading Node.js " + nodeVersion + " — this is a one-time 28 MB download…")
		if err := downloadAndExtractNode(emit); err != nil {
			emit("error:Node.js installation failed: " + err.Error())
			return types.ClaudeCodeStatus{Error: "Node.js install failed: " + err.Error()}
		}
		npmPath = bundledNpmBin()
		emit("node:done")
	} else {
		emit("node:done")
	}

	// ── Step 2: Install Claude Code CLI ─────────────────────────────────────
	if !managedClaudeIsPinned() {
		emit("step:claude")
		emit("Installing Claude Code CLI — this takes about a minute…")
		if err := runNpmInstall(npmPath, claudeInstallDir(), "@anthropic-ai/claude-code@"+claudeCodeVersion); err != nil {
			emit("error:Claude Code installation failed: " + err.Error())
			return types.ClaudeCodeStatus{Error: "Claude Code install failed: " + err.Error()}
		}
		emit("claude:done")
	} else {
		emit("claude:done")
	}

	emit("step:auth")
	return a.CheckClaudeCode()
}

// SetupCodexCLI orchestrates OpenAI Codex CLI setup for ChatGPT-account users,
// through the same pinned-Node pipeline as Claude Code. Progress streams over
// the shared "setup:progress" channel (one setup wizard runs at a time).
func (a *App) SetupCodexCLI() types.ClaudeCodeStatus {
	emit := func(msg string) {
		runtime.EventsEmit(a.ctx, "setup:progress", msg)
	}

	npmPath := bundledNpmBin()
	if _, err := os.Stat(npmPath); err != nil {
		emit("step:node")
		emit("Downloading Node.js " + nodeVersion + " — this is a one-time 28 MB download…")
		if err := downloadAndExtractNode(emit); err != nil {
			emit("error:Node.js installation failed: " + err.Error())
			return types.ClaudeCodeStatus{Error: "Node.js install failed: " + err.Error()}
		}
		npmPath = bundledNpmBin()
	}
	emit("node:done")

	if !managedCodexIsPinned() {
		emit("step:claude")
		emit("Installing Codex CLI — this takes about a minute…")
		if err := runNpmInstall(npmPath, codexInstallDir(), "@openai/codex@"+codexCLIVersion); err != nil {
			emit("error:Codex installation failed: " + err.Error())
			return types.ClaudeCodeStatus{Error: "Codex install failed: " + err.Error()}
		}
	}
	emit("claude:done")

	emit("step:auth")
	return a.CheckCodexCLI()
}

func runNpmInstall(npmPath, prefix, pkg string) error {
	_ = os.MkdirAll(prefix, 0700)
	_ = os.Chmod(prefix, 0700)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := cmdExec(ctx, npmPath, "install", "-g", pkg, "--prefix", prefix)
	// Ensure our bundled node is on PATH so npm scripts can find it.
	cmd.Env = append(os.Environ(),
		"PATH="+nodeInstallBinDir()+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	platform.HideWindow(cmd)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// ── Node.js download + extraction ───────────────────────────────────────────

func nodeDownloadURL() (url, filename string) {
	arch := "x64"
	if goruntime.GOARCH == "arm64" {
		arch = "arm64"
	}
	switch goruntime.GOOS {
	case "windows":
		filename = fmt.Sprintf("node-%s-win-%s.zip", nodeVersion, arch)
	case "darwin":
		filename = fmt.Sprintf("node-%s-darwin-%s.tar.gz", nodeVersion, arch)
	default:
		filename = fmt.Sprintf("node-%s-linux-%s.tar.gz", nodeVersion, arch)
	}
	url = "https://nodejs.org/dist/" + nodeVersion + "/" + filename
	return
}

func downloadAndExtractNode(emit func(string)) error {
	url, filename := nodeDownloadURL()
	wantSum, ok := nodeSHA256[filename]
	if !ok {
		return fmt.Errorf("no pinned checksum for %s on this platform; refusing to install unverified Node.js", filename)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxNodeDownload+1))
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	if len(data) > maxNodeDownload {
		return fmt.Errorf("download exceeds %d bytes; aborting", maxNodeDownload)
	}
	sum := sha256.Sum256(data)
	gotSum := hex.EncodeToString(sum[:])
	if gotSum != wantSum {
		return fmt.Errorf("checksum mismatch for %s: got %s, expected %s — refusing to install", filename, gotSum, wantSum)
	}
	emit("Extracting Node.js…")
	dest := nodeInstallDir()
	_ = os.MkdirAll(dest, 0755)
	if goruntime.GOOS == "windows" {
		return extractNodeZip(data, dest)
	}
	return extractNodeTarGz(data, dest)
}

// securePath joins rel under dest, rejecting absolute paths, volume prefixes,
// and any traversal that would escape dest.
func securePath(dest, rel string) (string, error) {
	rel = filepath.FromSlash(rel)
	// Reject absolute, drive-prefixed, and rooted paths (on Windows IsAbs is
	// false for `\etc` yet it still escapes any relative join semantics).
	if filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" ||
		(len(rel) > 0 && os.IsPathSeparator(rel[0])) {
		return "", fmt.Errorf("archive entry has absolute path: %q", rel)
	}
	target := filepath.Join(dest, rel)
	back, err := filepath.Rel(dest, target)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry escapes destination: %q", rel)
	}
	return target, nil
}

// maskMode strips setuid/setgid/sticky and any non-permission bits from an
// archive-supplied mode, flooring to a sane minimum.
func maskMode(mode int64) os.FileMode {
	m := os.FileMode(mode) & 0o777
	if m&0o400 == 0 {
		m |= 0o644
	}
	return m
}

// extractNodeZip extracts the Windows Node.js zip, stripping the top-level
// versioned directory. Entry paths are confined to dest and sizes are capped
// on actual bytes decompressed.
func extractNodeZip(data []byte, dest string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(r.File) > maxExtractEntries {
		return fmt.Errorf("archive has %d entries (limit %d)", len(r.File), maxExtractEntries)
	}
	var total int64
	for _, f := range r.File {
		// Strip first path component: "node-v22.14.0-win-x64/"
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		target, err := securePath(dest, parts[1])
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return err
		}
		n, cpErr := io.Copy(out, io.LimitReader(rc, maxExtractEntry+1))
		_ = out.Close()
		_ = rc.Close()
		if cpErr != nil {
			return cpErr
		}
		if n > maxExtractEntry {
			return fmt.Errorf("entry %q exceeds %d bytes", f.Name, maxExtractEntry)
		}
		total += n
		if total > maxExtractTotal {
			return fmt.Errorf("archive exceeds %d bytes total", maxExtractTotal)
		}
	}
	return nil
}

// extractNodeTarGz extracts the macOS/Linux Node.js tarball, stripping the
// top-level directory. Paths are confined to dest, symlinks must resolve
// inside dest (Node's Unix layout needs relative in-tree links like
// bin/npm -> ../lib/node_modules/npm/bin/npm-cli.js), hardlinks are rejected,
// and mode bits are masked.
func extractNodeTarGz(data []byte, dest string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()
	tr := tar.NewReader(gr)
	var total int64
	entries := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if entries > maxExtractEntries {
			return fmt.Errorf("archive has more than %d entries", maxExtractEntries)
		}
		// Strip top-level component: "node-v22.14.0-darwin-arm64/"
		slash := strings.Index(hdr.Name, "/")
		if slash < 0 {
			continue
		}
		rel := hdr.Name[slash+1:]
		if rel == "" {
			continue
		}
		target, err := securePath(dest, rel)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, maskMode(hdr.Mode)|0o700)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, maskMode(hdr.Mode))
			if err != nil {
				return err
			}
			n, cpErr := io.Copy(f, io.LimitReader(tr, maxExtractEntry+1))
			_ = f.Close()
			if cpErr != nil {
				return cpErr
			}
			if n > maxExtractEntry {
				return fmt.Errorf("entry %q exceeds %d bytes", hdr.Name, maxExtractEntry)
			}
			total += n
			if total > maxExtractTotal {
				return fmt.Errorf("archive exceeds %d bytes total", maxExtractTotal)
			}
		case tar.TypeSymlink:
			if filepath.IsAbs(hdr.Linkname) || filepath.VolumeName(hdr.Linkname) != "" {
				return fmt.Errorf("symlink %q has absolute target %q", hdr.Name, hdr.Linkname)
			}
			// The resolved link target must stay inside dest.
			resolved := filepath.Join(filepath.Dir(target), filepath.FromSlash(hdr.Linkname))
			if back, err := filepath.Rel(dest, resolved); err != nil || back == ".." ||
				strings.HasPrefix(back, ".."+string(filepath.Separator)) {
				return fmt.Errorf("symlink %q escapes destination (target %q)", hdr.Name, hdr.Linkname)
			}
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %q: %w", hdr.Name, err)
			}
		case tar.TypeLink:
			return fmt.Errorf("hardlink %q not permitted in archive", hdr.Name)
		}
	}
	return nil
}
