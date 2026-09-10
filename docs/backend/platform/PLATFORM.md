# Platform Package

`internal/platform/` provides platform-specific utilities.

## Functions

### HideWindow(cmd *exec.Cmd)
Configures a command to run without a visible console window.

**Windows** (`hidewindow_windows.go`):
```go
func HideWindow(cmd *exec.Cmd) {
    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{}
    }
    cmd.SysProcAttr.HideWindow = true
    cmd.SysProcAttr.CreationFlags |= 0x08000000
}
```
Prevents black console windows from flashing when spawning external processes like the Claude CLI while preserving process attributes configured by the batch-command helper.

### BatchCommand(ctx, path, args...)

On Windows (`batch_windows.go`), `BatchCommand` constructs the required `cmd.exe /S /C` command line for the rare case where a `.cmd` or `.bat` shim must be used; `batch_other.go` provides the non-Windows implementation. Claude Code normally bypasses this path by launching its native executable or JavaScript entry point directly, which avoids command parsing failures when installation paths contain spaces.

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
