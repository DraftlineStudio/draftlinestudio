package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Pinned LTS version. Can be bumped in future releases.
const nodeVersion = "v22.14.0"

// ── Path helpers ────────────────────────────────────────────────────────────

func draftlineDataDir() string {
	dir, _ := os.UserConfigDir()
	p := filepath.Join(dir, "draftline")
	_ = os.MkdirAll(p, 0755)
	return p
}

// nodeInstallDir is where Draftline extracts a portable Node.js distribution.
func nodeInstallDir() string {
	return filepath.Join(draftlineDataDir(), "node")
}

// npmGlobalDir is where we install npm global packages (including claude).
func npmGlobalDir() string {
	return filepath.Join(draftlineDataDir(), "npm-global")
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
	d := npmGlobalDir()
	if goruntime.GOOS == "windows" {
		return filepath.Join(d, "claude.cmd")
	}
	return filepath.Join(d, "bin", "claude")
}

// resolveNpmBin returns the best available npm binary (system PATH → bundled), or "".
func resolveNpmBin() string {
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	p := bundledNpmBin()
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// cmdExec wraps exec.CommandContext so that on Windows, .cmd and .bat files
// are invoked through cmd.exe, which correctly handles paths that contain spaces.
func cmdExec(ctx context.Context, path string, args ...string) *exec.Cmd {
	if goruntime.GOOS == "windows" {
		lp := strings.ToLower(path)
		if strings.HasSuffix(lp, ".cmd") || strings.HasSuffix(lp, ".bat") {
			return exec.CommandContext(ctx, "cmd.exe", append([]string{"/C", path}, args...)...)
		}
	}
	return exec.CommandContext(ctx, path, args...)
}

// claudeExec builds an exec.Cmd to run Claude Code. On Windows it calls node.exe
// + cli.js directly, completely bypassing cmd.exe and its argument-quoting quirks.
// On macOS/Linux the shell script is directly executable so cmdExec is used as-is.
func claudeExec(ctx context.Context, claudePath string, args ...string) *exec.Cmd {
	if goruntime.GOOS != "windows" {
		return cmdExec(ctx, claudePath, args...)
	}

	// npm puts cli.js in node_modules/@anthropic-ai/claude-code/ inside the global prefix dir.
	// claude.cmd lives in that same prefix dir, so dirname(claude.cmd) is the prefix.
	cliJS := filepath.Join(filepath.Dir(claudePath), "node_modules", "@anthropic-ai", "claude-code", "cli.js")
	if _, err := os.Stat(cliJS); err != nil {
		// cli.js not found at expected location — fall back to cmd.exe approach
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

// resolveClaudeBin returns the best available claude binary (system PATH → bundled), or "".
func resolveClaudeBin() string {
	if p, err := exec.LookPath("claude"); err == nil {
		return p
	}
	p := bundledClaudeBin()
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// ── Setup orchestration ─────────────────────────────────────────────────────

// SetupClaudeCode orchestrates the full Claude Code setup in one call:
//  1. Download + extract portable Node.js if npm is unavailable
//  2. npm install -g @anthropic-ai/claude-code --prefix {npmGlobalDir}
//
// Progress is streamed to the frontend via "setup:progress" events.
// The function blocks until setup is complete (or fails).
func (a *App) SetupClaudeCode() ClaudeCodeStatus {
	emit := func(msg string) {
		runtime.EventsEmit(a.ctx, "setup:progress", msg)
	}

	// ── Step 1: Ensure npm is available ─────────────────────────────────────
	npmPath := resolveNpmBin()
	if npmPath == "" {
		emit("step:node")
		emit("Downloading Node.js " + nodeVersion + " — this is a one-time 28 MB download…")
		if err := downloadAndExtractNode(emit); err != nil {
			emit("error:Node.js installation failed: " + err.Error())
			return ClaudeCodeStatus{Error: "Node.js install failed: " + err.Error()}
		}
		npmPath = bundledNpmBin()
		emit("node:done")
	} else {
		emit("node:done")
	}

	// ── Step 2: Install Claude Code CLI ─────────────────────────────────────
	claudePath := resolveClaudeBin()
	if claudePath == "" {
		emit("step:claude")
		emit("Installing Claude Code CLI — this takes about a minute…")
		if err := runNpmInstall(npmPath); err != nil {
			emit("error:Claude Code installation failed: " + err.Error())
			return ClaudeCodeStatus{Error: "Claude Code install failed: " + err.Error()}
		}
		emit("claude:done")
	} else {
		emit("claude:done")
	}

	emit("step:auth")
	return a.CheckClaudeCode()
}

func runNpmInstall(npmPath string) error {
	prefix := npmGlobalDir()
	_ = os.MkdirAll(prefix, 0755)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := cmdExec(ctx, npmPath, "install", "-g", "@anthropic-ai/claude-code", "--prefix", prefix)
	// Ensure our bundled node is on PATH so npm scripts can find it.
	cmd.Env = append(os.Environ(),
		"PATH="+nodeInstallBinDir()+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	hideWindow(cmd)
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
	url, _ := nodeDownloadURL()
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	emit("Extracting Node.js…")
	dest := nodeInstallDir()
	_ = os.MkdirAll(dest, 0755)
	if goruntime.GOOS == "windows" {
		return extractNodeZip(data, dest)
	}
	return extractNodeTarGz(data, dest)
}

// extractNodeZip extracts the Windows Node.js zip, stripping the top-level versioned directory.
func extractNodeZip(data []byte, dest string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		// Strip first path component: "node-v22.14.0-win-x64/"
		parts := strings.SplitN(filepath.ToSlash(f.Name), "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(parts[1]))
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
			rc.Close()
			return err
		}
		_, cpErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if cpErr != nil {
			return cpErr
		}
	}
	return nil
}

// extractNodeTarGz extracts the macOS/Linux Node.js tarball, stripping the top-level directory.
func extractNodeTarGz(data []byte, dest string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
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
		target := filepath.Join(dest, rel)
		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, cpErr := io.Copy(f, tr)
			f.Close()
			if cpErr != nil {
				return cpErr
			}
		case tar.TypeSymlink:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Remove(target)
			_ = os.Symlink(hdr.Linkname, target)
		}
	}
	return nil
}
