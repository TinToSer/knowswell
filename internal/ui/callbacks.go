// callbacks.go contains syscall.NewCallback wrappers for all Win32 WndProc
// functions. Callbacks must be created at package init time (not inside
// functions) and must be kept alive for the lifetime of the process.
package ui

import "syscall"

// These variables hold the final uintptr callback handles that are
// passed to RegisterClassExW. They are created once at init time.
var (
	toolbarWndProc  = syscall.NewCallback(toolbarWndProcImpl)
	chatBarWndProc  = syscall.NewCallback(chatBarWndProcImpl)
	snipWndProc     = syscall.NewCallback(snipWndProcImpl)
	overlayWndProc  = syscall.NewCallback(overlayWndProcImpl)
	settingsWndProc = syscall.NewCallback(settingsWndProcImpl)
)
