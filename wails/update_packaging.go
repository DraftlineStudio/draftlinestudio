package main

// Linux packaging is not one thing: the same release ships an AppImage, a
// .deb, and an .rpm, and the running Draftline has to work out which of them
// it is before it can offer the matching update or apply it.
//
//   AppImage  the runtime exports $APPIMAGE; the update is a new file swapped
//             into the old one's place, then relaunched.
//   deb/rpm   installed under /usr or /opt by the system package manager; the
//             update is applied by that package manager in a terminal window
//             the writer can watch (it will ask for their password), which
//             then starts the new Draftline.
//
// Nothing here is reachable on other platforms; the helpers are portable Go
// so the package builds everywhere.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"draftline/internal/fsutil"
	"draftline/internal/platform"
)

// linuxInstallKind reports "appimage", "deb", "rpm", or "" for a Draftline
// that was built from source or unpacked by hand (no in-app update path).
func linuxInstallKind() string {
	if os.Getenv("APPIMAGE") != "" {
		return "appimage"
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	if !strings.HasPrefix(exe, "/usr/") && !strings.HasPrefix(exe, "/opt/") {
		return ""
	}
	if hasCommand("dpkg") && (hasCommand("apt-get") || hasCommand("apt")) {
		return "deb"
	}
	if hasCommand("rpm") && (hasCommand("dnf") || hasCommand("yum") || hasCommand("zypper")) {
		return "rpm"
	}
	return ""
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// linuxAssetSuffix maps an install kind to the release asset that replaces it.
func linuxAssetSuffix(kind string) string {
	switch kind {
	case "appimage":
		return "-linux-x86_64.AppImage"
	case "deb":
		return "-linux-x86_64.deb"
	case "rpm":
		return "-linux-x86_64.rpm"
	}
	return ""
}

// applyLinuxUpdate installs a verified download and starts the new
// Draftline. It reports whether that was set in motion; when it was, the
// caller quits this instance so the replacement can take over.
func applyLinuxUpdate(kind, path string) bool {
	switch kind {
	case "appimage":
		return replaceAppImage(os.Getenv("APPIMAGE"), path)
	case "deb", "rpm":
		return runPackageInstallInTerminal(path)
	}
	return false
}

// replaceAppImage swaps the new AppImage into the running one's place. The
// running image stays mounted from its old inode, so the rename is safe, and
// CopyFileAtomic stages beside the destination so the swap holds even when the
// download cache is on another filesystem.
func replaceAppImage(current, downloaded string) bool {
	if current == "" {
		return false
	}
	if err := fsutil.CopyFileAtomic(downloaded, current, 0o755); err != nil {
		return false
	}
	_ = os.Remove(downloaded)
	cmd := exec.Command(current)
	platform.Detach(cmd)
	return cmd.Start() == nil
}

// terminalLaunchers lists terminal emulators with the flag that runs a
// command, most-portable first. x-terminal-emulator is the Debian/Ubuntu
// alternative that resolves to whatever the desktop installed.
var terminalLaunchers = [][]string{
	{"x-terminal-emulator", "-e"},
	{"gnome-terminal", "--"},
	{"konsole", "-e"},
	{"xfce4-terminal", "-e"},
	{"mate-terminal", "-e"},
	{"tilix", "-e"},
	{"kitty"},
	{"alacritty", "-e"},
	{"xterm", "-e"},
}

// runPackageInstallInTerminal writes a small shell script beside the
// download and opens it in a terminal window: the package manager installs
// the file (resolving dependencies itself), then the script starts the new
// Draftline. Failures leave the window open so the writer can read them.
func runPackageInstallInTerminal(pkg string) bool {
	script := filepath.Join(filepath.Dir(pkg), "apply-update.sh")
	if err := os.WriteFile(script, []byte(packageInstallScript(pkg)), 0o755); err != nil {
		return false
	}
	for _, launcher := range terminalLaunchers {
		bin, err := exec.LookPath(launcher[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, append(launcher[1:], script)...)
		platform.Detach(cmd)
		if cmd.Start() == nil {
			return true
		}
	}
	return false
}

func packageInstallScript(pkg string) string {
	return fmt.Sprintf(`#!/bin/sh
# Draftline update: installs the downloaded package with the system package
# manager, then starts the new version. Written by Draftline; safe to delete.
PKG=%q
echo "Draftline update: installing $(basename "$PKG")"
echo "Your password may be requested to authorise the installation."
echo
status=1
if command -v apt-get >/dev/null 2>&1; then
  sudo apt-get install -y "$PKG" && status=0
  [ "$status" -eq 0 ] || sudo apt-get install -f -y && status=0
elif command -v dnf >/dev/null 2>&1; then
  sudo dnf install -y "$PKG" && status=0
elif command -v yum >/dev/null 2>&1; then
  sudo yum install -y "$PKG" && status=0
elif command -v zypper >/dev/null 2>&1; then
  sudo zypper --non-interactive install --allow-unsigned-rpm "$PKG" && status=0
else
  echo "No supported package manager was found (apt-get, dnf, yum, or zypper)."
fi
echo
if [ "$status" -eq 0 ]; then
  echo "Draftline is up to date. Starting it now."
  rm -f "$PKG"
  sleep 1
  (setsid draftline >/dev/null 2>&1 &)
  sleep 2
else
  echo "The update did not complete. The package is still at:"
  echo "  $PKG"
  echo
  printf "Press Enter to close this window. "
  read -r _
fi
`, pkg)
}
