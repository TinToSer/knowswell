// Toolbar window: a small always-on-top pill at the top of the screen
// that can be expanded to reveal the action buttons.
package ui

import (
	"fmt"
	"runtime"
	"unsafe"
)

// Toolbar layout constants.
const (
	tbCollapsedW = 48
	tbExpandedW  = 380
	tbHeight     = 48
	btnSize      = 36
)

// Button IDs (used as hit-test return values; not Win32 control IDs).
const (
	btnSnip     = 0
	btnAsk      = 1
	btnSettings = 2
	btnClose    = 3
	btnLogo     = 4
)

// toolbarWndProcImpl is the WndProc for the SAToolbar class.
func toolbarWndProcImpl(hwnd, umsg, wParam, lParam uintptr) (result uintptr) {
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
		paintToolbar(hwnd, a)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmNCHitTest:
		pt := point{X: int32(int16(loWord(lParam))), Y: int32(int16(hiWord(lParam)))}
		pScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
		if !a.tbExpanded {
			return htCaption
		}
		if a.toolbarHitTest(int(pt.X), int(pt.Y)) >= 0 {
			return htClient
		}
		return htCaption
	case wmNCLButtonDown:
		if !a.tbExpanded {
			x0, y0 := a.tbX, a.tbY
			defProc(hwnd, umsg, wParam, lParam)
			dx, dy := a.tbX-x0, a.tbY-y0
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			if dx+dy < 5 {
				a.expandToolbar()
			}
			return 0
		}
	case wmLButtonDown:
		x, y := loWord(lParam), hiWord(lParam)
		a.toolbarClick(hwnd, int(x), int(y))
		return 0
	case wmMouseMove:
		x, y := loWord(lParam), hiWord(lParam)
		btn := a.toolbarHitTest(int(x), int(y))
		if btn != a.hoverBtn {
			a.hoverBtn = btn
			invalidate(HWND(hwnd))
		}
		trackMouse(HWND(hwnd))
		return 0
	case wmMouseLeave:
		a.hoverBtn = -1
		invalidate(HWND(hwnd))
		return 0
	case wmTimer:
		switch wParam {
		case 999:
			pKillTimer.Call(uintptr(hwnd), 999)
			if err := a.AddSnipFromCapture(
				a.snipStartX, a.snipStartY,
				a.snipEndX, a.snipEndY,
			); err == nil {
				if atts := a.Attachments(); len(atts) > 0 {
					copyImageToClipboard(atts[len(atts)-1].Image)
				}
				showChatBar(a)
			}
		case 998:
			a.tbAnimFrame++
			if !a.tbExpanded {
				invalidate(a.Toolbar)
			}
		}
		return 0
	case wmMove:
		var wr rect
		pGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&wr)))
		a.tbX = int(wr.Left)
		a.tbY = int(wr.Top)
		a.cfg.ToolbarX = a.tbX
		a.cfg.ToolbarY = a.tbY
		_ = a.cfg.Save()
		return 0
	case wmDestroy:
		a.cancelLLM()
		pPostQuitMessage.Call(0)
		return 0
	}
	return defProc(hwnd, umsg, wParam, lParam)
}

// currentApp returns the singleton App or nil if Run hasn't been called.
var globalApp *App

func currentApp() *App { return globalApp }

func defProc(hwnd, msg, w, l uintptr) uintptr {
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, w, l)
	return r
}

// ---------- Layout ----------

type buttonRect struct {
	x, y, w, h int
	id         int
	label      string
	clr        uintptr
}

// Toolbar button widths (height is always btnSize).
const (
	snipBtnW  = 54
	askBtnW   = 46
	settBtnW  = 64
	closeBtnW = 64
)

func (a *App) toolbarButtons() []buttonRect {
	if !a.tbExpanded {
		return []buttonRect{{
			x: 6, y: 4, w: tbCollapsedW - 12, h: tbHeight - 8,
			id: -1, label: "≡", clr: clrSurface,
		}}
	}
	y := (tbHeight - btnSize) / 2
	logoSz := tbHeight - 16
	logoX := tbExpandedW/2 - logoSz/2
	closeX := tbExpandedW - closeBtnW - 6
	settX := closeX - 6 - settBtnW
	return []buttonRect{
		{x: 6, y: y, w: snipBtnW, h: btnSize, id: btnSnip, label: "[SNIP]", clr: clrSnip},
		{x: 6 + snipBtnW + 6, y: y, w: askBtnW, h: btnSize, id: btnAsk, label: "[ASK]", clr: clrAccent},
		{x: logoX, y: (tbHeight - logoSz) / 2, w: logoSz, h: logoSz, id: btnLogo, label: "", clr: 0},
		{x: settX, y: y, w: settBtnW, h: btnSize, id: btnSettings, label: "[CONFIG]", clr: clrSurface},
		{x: closeX, y: y, w: closeBtnW, h: btnSize, id: btnClose, label: "[CLOSE]", clr: clrDanger},
	}
}

func (a *App) toolbarHitTest(x, y int) int {
	for _, b := range a.toolbarButtons() {
		if x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			return b.id
		}
	}
	return -1
}

func (a *App) toolbarClick(hwnd uintptr, x, y int) {
	if !a.tbExpanded {
		a.expandToolbar()
		return
	}
	switch a.toolbarHitTest(x, y) {
	case btnSnip:
		a.startSnip()
	case btnAsk:
		showChatBar(a)
	case btnSettings:
		showSettings(a)
	case btnClose:
		a.collapseToolbar()
	case btnLogo:
		openURL("https://github.com/tintoser")
	}
}

func (a *App) expandToolbar() {
	a.tbExpanded = true
	a.tbW = tbExpandedW
	pKillTimer.Call(uintptr(a.Toolbar), 998)
	pMoveWindow.Call(
		uintptr(a.Toolbar),
		uintptr(a.tbX), uintptr(a.tbY),
		uintptr(tbExpandedW), uintptr(tbHeight), 1,
	)
	invalidate(a.Toolbar)
}

func (a *App) collapseToolbar() {
	a.tbExpanded = false
	a.tbW = tbCollapsedW
	hideChatBar(a)
	hideOverlay(a)
	pSetTimer.Call(uintptr(a.Toolbar), 998, 50, 0)
	pMoveWindow.Call(
		uintptr(a.Toolbar),
		uintptr(a.tbX), uintptr(a.tbY),
		uintptr(tbCollapsedW), uintptr(tbHeight), 1,
	)
	invalidate(a.Toolbar)
}

// ---------- Painting ----------

func paintToolbar(hwnd uintptr, a *App) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w := int(rc.Right)
	h := int(rc.Bottom)

	memDC, _, flush := beginDoubleBuffer(hdc, w, h)
	defer flush()

	// Dark retro background.
	drawRoundRect(memDC, 0, 0, w, h, 3, clrBackground, clrBorder)

	if !a.tbExpanded {
		// Blinking phosphor border — alternates bright/dim every 4 frames (200 ms).
		var glowClr uintptr
		if (a.tbAnimFrame/4)%2 == 0 {
			glowClr = clrAccent
		} else {
			glowClr = clrBorder
		}
		drawRoundRect(memDC, 0, 0, w, h, 3, glowClr, 0)
		drawRoundRect(memDC, 3, 3, w-6, h-6, 2, clrBackground, 0)

		// ">_" retro terminal prompt icon.
		drawCenteredText(memDC, ">_", 3, 3, w-6, h-6, clrAccent, 15, true)
		return
	}

	// Expanded: draw buttons.
	for _, b := range a.toolbarButtons() {
		if b.id == btnLogo {
			if a.hoverBtn == btnLogo {
				drawRoundRect(memDC, b.x-3, b.y-3, b.w+6, b.h+6, 2, clrSurface2, clrBorder)
			}
			if logoLoaded() {
				drawLogo(memDC, b.x, b.y, b.w, b.h)
			} else {
				drawCenteredText(memDC, ">_", b.x, b.y, b.w, b.h, clrAccent, 14, true)
			}
			continue
		}
		clr := b.clr
		if a.hoverBtn == b.id {
			switch b.id {
			case btnSnip:
				clr = clrSnipHover
			case btnAsk:
				clr = clrAccentHover
			case btnClose:
				clr = clrDangerHover
			default:
				clr = clrSurface2
			}
		}
		drawRoundRect(memDC, b.x, b.y, b.w, b.h, 2, clr, 0)

		var tClr uintptr
		var tSz int
		switch b.id {
		case btnSnip:
			tClr, tSz = rgb(30, 18, 0), 10 // dark text on amber
		case btnAsk:
			tClr, tSz = rgb(0, 25, 8), 11 // dark text on green
		case btnSettings:
			tClr, tSz = clrAccent, 10 // [CONFIG] on dark surface
		case btnClose:
			tClr, tSz = rgb(25, 0, 0), 10 // dark text on red
		default:
			tClr, tSz = clrText, 11
		}
		drawCenteredText(memDC, b.label, b.x, b.y, b.w, b.h, tClr, tSz, true)
	}
}

// ---------- Snip workflow ----------

func (a *App) startSnip() {
	if a.Snip != 0 {
		return
	}
	// Reset all coordinates so previous snip outline is not shown.
	a.snipStartX, a.snipStartY, a.snipEndX, a.snipEndY = 0, 0, 0, 0
	a.snipDragging = false

	vx, vy, vw, vh := virtualScreenBounds()
	h, _, _ := pCreateWindowExW.Call(
		wsExTopMost|wsExLayered|wsExToolWindow,
		uintptr(unsafe.Pointer(utf16Ptr("SASnip"))),
		0,
		wsPopup|wsVisible,
		uintptr(vx), uintptr(vy), uintptr(vw), uintptr(vh),
		0, 0, uintptr(hInstance()), 0,
	)
	a.Snip = HWND(h)
	pSetLayeredWindowAttributes.Call(uintptr(h), snipKeyColor, 200, lwaAlpha|lwaColorKey)
	pSetWindowDisplayAffinity.Call(uintptr(h), wdaExcludeFromCapture)
	pShowWindow.Call(uintptr(h), swShow)
	pUpdateWindow.Call(uintptr(h))
	cursor, _, _ := pLoadCursorW.Call(0, idcCross)
	pSetCursor.Call(cursor)
	pSetCapture.Call(uintptr(h))
}

func (a *App) finishSnip() {
	x1 := intMin(a.snipStartX, a.snipEndX)
	y1 := intMin(a.snipStartY, a.snipEndY)
	w := intMax(a.snipStartX, a.snipEndX) - x1
	h := intMax(a.snipStartY, a.snipEndY) - y1
	if w > 5 && h > 5 {
		a.snipStartX, a.snipStartY = x1, y1
		a.snipEndX, a.snipEndY = w, h
		pSetTimer.Call(uintptr(a.Toolbar), 999, 80, 0)
	} else {
		a.snippingOff()
	}
}

func (a *App) snippingOff() {
	a.snipDragging = false
}

func virtualScreenBounds() (x, y, w, h int) {
	vx, _, _ := pGetSystemMetrics.Call(smXVirtScreen)
	vy, _, _ := pGetSystemMetrics.Call(smYVirtScreen)
	vw, _, _ := pGetSystemMetrics.Call(smCXVirtScreen)
	vh, _, _ := pGetSystemMetrics.Call(smCYVirtScreen)
	return int(int32(vx)), int(int32(vy)), int(vw), int(vh)
}

// ---------- Run() entry point ----------

// Run registers all window classes, creates the toolbar, and enters
// the Win32 message loop. It returns once WM_QUIT is received.
func (a *App) Run() error {
	runtime.LockOSThread() // Win32 UI must stay on one OS thread
	defer func() { recover() }()

	globalApp = a
	a.registerClasses()
	_ = msftedit.Load() // pre-load so RICHEDIT50W class is registered early
	go loadLogo()

	sw, sh := screenSize()
	_ = sh

	a.tbX = intMin(a.cfg.ToolbarX, sw-tbCollapsedW)
	if a.tbX < 0 {
		a.tbX = 200
	}
	a.tbY = a.cfg.ToolbarY
	if a.tbY < 0 {
		a.tbY = 8
	}
	a.tbW = tbCollapsedW
	a.tbH = tbHeight
	a.hoverBtn = -1

	clsName := utf16Ptr("SAToolbar")
	winTitle := utf16Ptr("KnowsWell")
	h, _, lastErr := pCreateWindowExW.Call(
		wsExTopMost|wsExLayered|wsExToolWindow,
		uintptr(unsafe.Pointer(clsName)),
		uintptr(unsafe.Pointer(winTitle)),
		wsPopup,
		uintptr(a.tbX), uintptr(a.tbY),
		uintptr(a.tbW), uintptr(a.tbH),
		0, 0, uintptr(hInstance()), 0,
	)
	runtime.KeepAlive(clsName)
	runtime.KeepAlive(winTitle)

	if h == 0 {
		return fmt.Errorf("CreateWindowExW failed: %v", lastErr)
	}
	a.Toolbar = HWND(h)

	noRound := uint32(1) // DWMWCP_DONOTROUND — sharp retro corners on Win11
	pDwmSetWindowAttribute.Call(uintptr(h), 33, uintptr(unsafe.Pointer(&noRound)), 4)
	pSetWindowDisplayAffinity.Call(uintptr(h), a.displayAffinity())
	pSetLayeredWindowAttributes.Call(uintptr(h), 0, uintptr(a.alphaValue()), lwaAlpha)

	pShowWindow.Call(uintptr(h), swShow)
	pUpdateWindow.Call(uintptr(h))
	setTopmost(a.Toolbar)

	pSetTimer.Call(uintptr(h), 998, 400, 0)

	var m msg
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func screenSize() (int, int) {
	w, _, _ := pGetSystemMetrics.Call(smCXScreen)
	h, _, _ := pGetSystemMetrics.Call(smCYScreen)
	return int(w), int(h)
}

// openURL opens a URL in the system default browser.
func openURL(url string) {
	pShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("open"))),
		uintptr(unsafe.Pointer(utf16Ptr(url))),
		0, 0, 1,
	)
}
