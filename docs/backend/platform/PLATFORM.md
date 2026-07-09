# Platform Package

`internal/platform/` provides platform-specific utilities.

## Functions

### HideWindow(cmd *exec.Cmd)
Configures a command to run without a visible console window.

**Windows** (`hidewindow_windows.go`):
```go
func HideWindow(cmd *exec.Cmd) {
    cmd.SysProcAttr = &syscall.SysProcAttr{
        HideWindow:    true,
        CreationFlags: 0x08000000, // CREATE_NO_WINDOW
    }
}
```
Prevents black console windows from flashing when spawning external processes like the Claude CLI.

**macOS/Linux** (`hidewindow_other.go`):
```go
func HideWindow(cmd *exec.Cmd) {} // no-op
```
No action needed - these platforms don't show console windows for background processes.

## Build Tags

Uses Go build tags to compile only the appropriate implementation:
- `//go:build windows` - Windows-specific code
- `//go:build !windows` - Everything else

## Usage

```go
import "draftline/internal/platform"

cmd := exec.Command("claude", "auth", "login")
platform.HideWindow(cmd)
cmd.Run()
```

## Used By

- `app.go` - Claude CLI authentication
- `setup.go` - Node.js and Claude Code installation
