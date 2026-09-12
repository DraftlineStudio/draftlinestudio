//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procShowWindow               = user32.NewProc("ShowWindow")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
)

const swRestore = 9

// FocusProcessWindow brings the given process's main visible window to the
// foreground (restoring it if minimised). Used when a book is already open
// in another Draftline instance: instead of a second copy, the author lands
// in the window that has it. Returns false when no visible window was found.
func FocusProcessWindow(pid int) bool {
	var target syscall.Handle
	cb := syscall.NewCallback(func(hwnd syscall.Handle, _ uintptr) uintptr {
		var windowPid uint32
		_, _, _ = procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowPid)))
		if int(windowPid) != pid {
			return 1 // continue enumeration
		}
		visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if visible == 0 {
			return 1
		}
		target = hwnd
		return 0 // stop
	})
	_, _, _ = procEnumWindows.Call(cb, 0)
	if target == 0 {
		return false
	}
	if iconic, _, _ := procIsIconic.Call(uintptr(target)); iconic != 0 {
		_, _, _ = procShowWindow.Call(uintptr(target), swRestore)
	}
	ok, _, _ := procSetForegroundWindow.Call(uintptr(target))
	return ok != 0
}
