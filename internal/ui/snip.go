// Snip window: a fullscreen translucent overlay that captures the
// user's mouse-drag selection. The captured rectangle is then passed
// back to the toolbar (via a short timer) which performs the screen
// capture and adds it as an attachment.
package ui

import (
	"unsafe"
)

// snipWndProcImpl is the WndProc implementation for the SASnip class.
func snipWndProcImpl(hwnd, umsg, wParam, lParam uintptr) (result uintptr) {
	defer func() {
		if recover() != nil {
			result = defProc(hwnd, umsg, wParam, lParam)
		}
	}()
	a := currentApp()
	if a == nil {
		return defProc(hwnd, umsg, wParam, lParam)
	}

	switch umsg {
	case wmPaint:
		paintSnip(hwnd, a)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmLButtonDown:
		a.snipDragging = true
		x, y := loWord(lParam), hiWord(lParam)
		pt := point{X: x, Y: y}
		pClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
		a.snipStartX, a.snipStartY = int(pt.X), int(pt.Y)
		a.snipEndX, a.snipEndY = a.snipStartX, a.snipStartY
		return 0
	case wmMouseMove:
		if a.snipDragging {
			x, y := loWord(lParam), hiWord(lParam)
			pt := point{X: x, Y: y}
			pClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
			a.snipEndX, a.snipEndY = int(pt.X), int(pt.Y)
			invalidate(HWND(hwnd))
		}
		return 0
	case wmLButtonUp:
		if a.snipDragging {
			a.snipDragging = false
			x, y := loWord(lParam), hiWord(lParam)
			pt := point{X: x, Y: y}
			pClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
			a.snipEndX, a.snipEndY = int(pt.X), int(pt.Y)
			pReleaseCapture.Call()
			pDestroyWindow.Call(uintptr(hwnd))
			a.Snip = 0
			a.finishSnip()
		}
		return 0
	case wmKeyDown:
		if wParam == vkEscape {
			pReleaseCapture.Call()
			pDestroyWindow.Call(uintptr(hwnd))
			a.Snip = 0
			a.snippingOff()
		}
		return 0
	case wmDestroy:
		a.Snip = 0
		return 0
	}
	return defProc(hwnd, umsg, wParam, lParam)
}

// paintSnip paints a dimmed fullscreen rectangle with a clear hole
// around the current selection.
func paintSnip(hwnd uintptr, a *App) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)

	memDC, _, flush := beginDoubleBuffer(hdc, w, h)
	defer flush()

	// Dim background.
	fillRect(memDC, 0, 0, w, h, rgb(0, 0, 0))

	if a.snipDragging || (a.snipEndX != a.snipStartX && a.snipEndY != a.snipStartY) {
		var wr rect
		pGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
		ox, oy := int(wr.Left), int(wr.Top)

		x1 := intMin(a.snipStartX, a.snipEndX) - ox
		y1 := intMin(a.snipStartY, a.snipEndY) - oy
		x2 := intMax(a.snipStartX, a.snipEndX) - ox
		y2 := intMax(a.snipStartY, a.snipEndY) - oy

		// Fill the selection with the color key so the layered window makes
		// those pixels transparent, revealing the screen underneath.
		fillRect(memDC, x1, y1, x2-x1, y2-y1, snipKeyColor)

		// 2px bright-yellow border around the transparent hole.
		bright := rgb(255, 220, 0)
		pen, _, _ := pCreatePen.Call(psSolid, 2, bright)
		nullBr, _, _ := pGetStockObject.Call(nullBrush)
		oldP := selectInto(memDC, uintptr(pen))
		oldB := selectInto(memDC, uintptr(nullBr))
		pRectangle.Call(uintptr(memDC),
			uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
		pSelectObject.Call(uintptr(memDC), oldP)
		pSelectObject.Call(uintptr(memDC), oldB)
		pDeleteObject.Call(pen)

		// Size label above the selection.
		label := formatSize(x2-x1, y2-y1)
		drawCenteredText(memDC, label, x1, intMax(0, y1-22), x2-x1, 20,
			bright, 12, true)
	}

	drawCenteredText(memDC, "Drag to select  •  ESC to cancel",
		0, h-30, w, 20, rgb(210, 210, 230), 12, false)
}

func formatSize(w, h int) string {
	const digits = "0123456789"
	_ = digits
	return itoa(w) + " × " + itoa(h)
}

// itoa is a tiny int-to-string helper that avoids importing strconv
// for this one use site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
