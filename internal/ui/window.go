// Window-class registration helper plus the global app helpers
// (topmost positioning, repaint, mouse-tracking).
package ui

import (
	"unsafe"
)

// windowClasses lists the names of every Win32 class this package registers.
var windowClasses = []string{
	"SAToolbar",
	"SAChatBar",
	"SASnip",
	"SAOverlay",
	"SASettings",
}

// registerClasses registers all window classes used by the app.
func (a *App) registerClasses() {
	icc := initCommonControlsEx{DwSize: 8, DwICC: 0x00004004} // ICC_STANDARD_CLASSES | ICC_BAR_CLASSES
	pInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	for _, cls := range []struct {
		name string
		proc uintptr
	}{
		{"SAToolbar", toolbarWndProc},
		{"SAChatBar", chatBarWndProc},
		{"SASnip", snipWndProc},
		{"SAOverlay", overlayWndProc},
		{"SASettings", settingsWndProc},
	} {
		registerWndClass(cls.name, cls.proc)
	}
}

// registerWndClass registers a single window class.
func registerWndClass(name string, wndProc uintptr) {
	cursor, _, _ := pLoadCursorW.Call(0, idcArrow)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		Style:         0x0003, // CS_HREDRAW | CS_VREDRAW
		LpfnWndProc:   wndProc,
		HInstance:     hInstance(),
		LpszClassName: utf16Ptr(name),
		HCursor:       HCURSOR(cursor),
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
}

// setTopmost re-asserts that the given window is at the top of the z-order.
func setTopmost(hwnd HWND) {
	pSetWindowPos.Call(
		uintptr(hwnd), hwndTopMost,
		0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoActivate,
	)
}

// invalidate forces the window to repaint.
func invalidate(hwnd HWND) {
	pInvalidateRect.Call(uintptr(hwnd), 0, 1)
	pUpdateWindow.Call(uintptr(hwnd))
}

// trackMouse requests WM_MOUSELEAVE notifications for the window.
func trackMouse(hwnd HWND) {
	tme := trackMouseEvent{
		CbSize:    uint32(unsafe.Sizeof(trackMouseEvent{})),
		DwFlags:   tmeLeave,
		HwndTrack: hwnd,
	}
	pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
