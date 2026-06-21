// Overlay window: a large translucent floating window that displays
// the streaming LLM response with basic markdown rendering.
package ui

import (
	"strings"
	"unsafe"
)

const (
	ovPad  = 16
	ovMaxW = 700
	ovMaxH = 500
)

// overlayWndProcImpl is the WndProc implementation for the SAOverlay class.
func overlayWndProcImpl(hwnd, umsg, wParam, lParam uintptr) (result uintptr) {
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
		paintOverlay(hwnd, a)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmNCHitTest:
		pt := point{X: int32(int16(loWord(lParam))), Y: int32(int16(hiWord(lParam)))}
		pScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
		x, y := int(pt.X), int(pt.Y)
		var rc rect
		pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
		w := int(rc.Right)
		// Close and Copy buttons → client click; everything else → draggable caption.
		if y >= 9 && y <= 33 && ((x >= w-70 && x <= w-16) || (x >= w-136 && x <= w-82)) {
			return htClient
		}
		return htCaption
	case wmLButtonDown:
		x, y := loWord(lParam), hiWord(lParam)
		a.overlayClick(hwnd, int(x), int(y))
		return 0
	case wmTimer:
		if wParam == 997 {
			pKillTimer.Call(uintptr(hwnd), 997)
			a.overlayCopied = false
			invalidate(HWND(hwnd))
		}
		return 0
	case wmSize:
		if a.hwRichEdit != 0 {
			var rc rect
			pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
			w, h := int(rc.Right), int(rc.Bottom)
			pMoveWindow.Call(uintptr(a.hwRichEdit),
				uintptr(ovPad), 44,
				uintptr(w-ovPad*2), uintptr(h-44-ovPad), 1)
		}
		return 0
	case wmDestroy:
		a.Overlay = 0
		a.hwRichEdit = 0
		return 0
	}
	return defProc(hwnd, umsg, wParam, lParam)
}

// showOverlay creates (if necessary) and updates the overlay with the
// given text. It is safe to call from the UI thread only.
func showOverlay(a *App, text string) {
	if a.Overlay == 0 {
		// Ensure msftedit.dll is loaded so "RICHEDIT50W" class is registered.
		_ = msftedit.Load()

		sw, sh := screenSize()
		w := intMin(ovMaxW, sw-100)
		h := intMin(ovMaxH, sh-200)
		x := (sw - w) / 2
		y := (sh - h) / 2

		hh, _, _ := pCreateWindowExW.Call(
			wsExTopMost|wsExLayered|wsExToolWindow,
			uintptr(unsafe.Pointer(utf16Ptr("SAOverlay"))),
			0,
			wsPopup|wsVisible,
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			0, 0, uintptr(hInstance()), 0,
		)
		a.Overlay = HWND(hh)
		noRound := uint32(1)
		pDwmSetWindowAttribute.Call(uintptr(hh), 33, uintptr(unsafe.Pointer(&noRound)), 4)
		pSetWindowDisplayAffinity.Call(uintptr(hh), a.displayAffinity())
		pSetLayeredWindowAttributes.Call(uintptr(hh), 0, uintptr(a.alphaValue()), lwaAlpha)

		// RichEdit for the response body.
		re, _, _ := pCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(utf16Ptr("RICHEDIT50W"))),
			0,
			wsChild|wsVisible|esMultiLine|esReadOnly|esAutoVScroll|wsVScroll,
			uintptr(ovPad), 44,
			uintptr(w-ovPad*2), uintptr(h-44-ovPad),
			uintptr(hh), 0, uintptr(hInstance()), 0,
		)
		a.hwRichEdit = HWND(re)
		pSendMessageW.Call(uintptr(re), emSetBkgndColor, 0, clrOverlayBG)
		pSendMessageW.Call(uintptr(re), wmSetFont,
			uintptr(getFont("Courier New", 12, false)), 1)

		pShowWindow.Call(uintptr(hh), swShow)
		pUpdateWindow.Call(uintptr(hh))
		setTopmost(a.Overlay)
	}

	if a.hwRichEdit != 0 {
		streamRTFToRichEdit(a.hwRichEdit, text)
	}
	invalidate(a.Overlay) // repaint header
}

func hideOverlay(a *App) {
	if a.Overlay != 0 {
		pShowWindow.Call(uintptr(a.Overlay), swHide)
	}
}

func (a *App) overlayClick(hwnd uintptr, x, y int) {
	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w := int(rc.Right)
	if x >= w-70 && x <= w-16 && y >= 9 && y <= 33 {
		pDestroyWindow.Call(uintptr(hwnd))
		a.Overlay = 0
		return
	}
	if x >= w-136 && x <= w-82 && y >= 9 && y <= 33 {
		text := richEditText(a.hwRichEdit)
		if text == "" {
			text = a.StreamText()
		}
		copyTextToClipboard(text)
		a.overlayCopied = true
		pSetTimer.Call(uintptr(hwnd), 997, 1000, 0)
		invalidate(HWND(hwnd))
	}
}

// ---------- Painting ----------

func paintOverlay(hwnd uintptr, a *App) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)

	memDC, _, flush := beginDoubleBuffer(hdc, w, h)
	defer flush()

	drawRoundRect(memDC, 0, 0, w, h, 2, clrOverlayBG, clrBorder)
	// Header strip (RichEdit sits below this at y=44).
	drawRoundRect(memDC, 0, 0, w, 44, 2, clrSurface, 0)
	fillRect(memDC, 0, 22, w, 22, clrSurface)
	drawText(memDC, ">> KNOWSWELL AI", ovPad, 10, w-150, 24, clrAccent, 12, true)
	drawRoundRect(memDC, w-70, 9, 54, 24, 2, clrDanger, 0)
	drawCenteredText(memDC, "[CLOSE]", w-70, 9, 54, 24, rgb(0, 0, 0), 10, true)
	if a.overlayCopied {
		drawRoundRect(memDC, w-136, 9, 54, 24, 2, clrAccent, 0)
		drawCenteredText(memDC, "[COPIED]", w-136, 9, 54, 24, rgb(0, 0, 0), 10, true)
	} else {
		drawRoundRect(memDC, w-136, 9, 54, 24, 2, clrSurface2, clrBorder)
		drawCenteredText(memDC, "[COPY]", w-136, 9, 54, 24, clrText, 10, true)
	}
	_ = h
}

// renderMarkdown draws the response text with very basic markdown-like
// formatting (headings, bullets, code blocks).
func renderMarkdown(hdc HDC, text string, x, y, w, h int) {
	lines := strings.Split(text, "\n")
	lineH := 20
	curY := y
	maxY := y + h

	for _, line := range lines {
		if curY+lineH > maxY {
			break
		}

		bold := false
		clr := clrText
		size := 13
		indent := 0

		switch {
		case strings.HasPrefix(line, "# "):
			line = line[2:]
			bold, size = true, 16
		case strings.HasPrefix(line, "## "):
			line = line[3:]
			bold, size = true, 14
		case strings.HasPrefix(line, "### "):
			line = line[4:]
			bold, size, clr = true, 13, clrAccent
		case strings.HasPrefix(line, "```"):
			drawRoundRect(hdc, x, curY, w, lineH, 4, clrSurface2, clrBorder)
			drawText(hdc, line, x+6, curY+2, w-12, lineH-4, clrSnip, 11, false)
			curY += lineH + 2
			continue
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			line = "• " + line[2:]
			indent = 8
		case strings.TrimSpace(line) == "":
			curY += lineH / 2
			continue
		}

		// Approximate character-per-line wrap.
		charsPerLine := (w - indent) / 7
		if charsPerLine < 10 {
			charsPerLine = 10
		}
		for _, wl := range wrapLine(line, charsPerLine) {
			if curY+lineH > maxY {
				break
			}
			drawText(hdc, wl, x+indent, curY, w-indent, lineH, clr, size, bold)
			curY += lineH + 2
		}
	}
}

// wrapLine breaks a single line of text at word boundaries so it fits
// within the given character width.
func wrapLine(text string, width int) []string {
	if width < 1 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var out []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len()+len(w)+1 > width && cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
