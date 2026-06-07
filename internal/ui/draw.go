// Custom GDI drawing helpers and a small font cache. All windows share
// these so the visual style stays consistent without a UI framework.
package ui

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// ---------- Theme colors (COLORREF — BGR packed) ----------

// Retro green phosphor terminal theme — high contrast, max emphasis.
var (
	clrBackground  = rgb(2, 8, 2)
	clrSurface     = rgb(8, 18, 8)
	clrSurface2    = rgb(14, 30, 14)
	clrAccent      = rgb(0, 255, 80)
	clrAccentHover = rgb(80, 255, 120)
	clrAccentPress = rgb(0, 210, 60)
	clrDanger      = rgb(230, 50, 50)
	clrDangerHover = rgb(255, 75, 65)
	clrText        = rgb(160, 255, 140)
	clrTextDim     = rgb(80, 160, 70)
	clrBorder      = rgb(40, 120, 44)
	clrSnip        = rgb(220, 185, 0)
	clrSnipHover   = rgb(255, 220, 40)
	clrOverlayBG   = rgb(2, 8, 2)
)

// clrInputText is used for text typed into EDIT controls — softer than the
// neon phosphor accent so it reads naturally without eye strain.
var clrInputText = rgb(235, 255, 230)

// snipKeyColor is used as the color-key for the layered snip window.
var snipKeyColor = rgb(1, 0, 1)

// ---------- Font cache ----------

var (
	fontMu    sync.Mutex
	fontCache = map[string]HFONT{}
)

func getFont(face string, size int, bold bool) HFONT {
	key := fmt.Sprintf("%s|%d|%v", face, size, bold)
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := fontCache[key]; ok {
		return f
	}
	lf := logFont{
		LfHeight:  int32(-size),
		LfQuality: 5, // CLEARTYPE_QUALITY
	}
	if bold {
		lf.LfWeight = 700
	} else {
		lf.LfWeight = 400
	}
	name := syscall.StringToUTF16(face)
	copy(lf.LfFaceName[:], name)
	h, _, _ := pCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	f := HFONT(h)
	fontCache[key] = f
	return f
}

// ---------- GDI helpers ----------

func selectInto(hdc HDC, obj uintptr) uintptr {
	r, _, _ := pSelectObject.Call(uintptr(hdc), obj)
	return r
}

// drawRoundRect paints a filled rounded rectangle with an optional 1px border.
func drawRoundRect(hdc HDC, x, y, w, h, r int, fillClr, borderClr uintptr) {
	br, _, _ := pCreateSolidBrush.Call(fillClr)
	defer pDeleteObject.Call(br)

	var pen uintptr
	if borderClr != 0 {
		p, _, _ := pCreatePen.Call(psSolid, 1, borderClr)
		pen = p
		defer pDeleteObject.Call(pen)
	} else {
		pen, _, _ = pGetStockObject.Call(nullBrush)
	}

	oldPen := selectInto(hdc, pen)
	oldBr := selectInto(hdc, br)
	defer pSelectObject.Call(uintptr(hdc), oldPen)
	defer pSelectObject.Call(uintptr(hdc), oldBr)

	pRoundRect.Call(
		uintptr(hdc),
		uintptr(x), uintptr(y),
		uintptr(x+w), uintptr(y+h),
		uintptr(r*2), uintptr(r*2),
	)
}

// drawText paints left-aligned, vertically centred text.
func drawText(hdc HDC, text string, x, y, w, h int, clr uintptr, size int, bold bool) {
	font := getFont("Courier New", size, bold)
	old := selectInto(hdc, uintptr(font))
	defer pSelectObject.Call(uintptr(hdc), old)

	pSetTextColor.Call(uintptr(hdc), clr)
	pSetBkMode.Call(uintptr(hdc), transparent)

	txt := syscall.StringToUTF16(text)
	rc := rect{int32(x), int32(y), int32(x + w), int32(y + h)}
	pDrawTextW.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&txt[0])),
		uintptr(len(txt)-1),
		uintptr(unsafe.Pointer(&rc)),
		dtLeft|dtVCenter|dtSingleLine|dtNoClip,
	)
}

// drawCenteredText paints text horizontally and vertically centred.
func drawCenteredText(hdc HDC, text string, x, y, w, h int, clr uintptr, size int, bold bool) {
	font := getFont("Courier New", size, bold)
	old := selectInto(hdc, uintptr(font))
	defer pSelectObject.Call(uintptr(hdc), old)

	pSetTextColor.Call(uintptr(hdc), clr)
	pSetBkMode.Call(uintptr(hdc), transparent)

	txt := syscall.StringToUTF16(text)
	rc := rect{int32(x), int32(y), int32(x + w), int32(y + h)}
	pDrawTextW.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&txt[0])),
		uintptr(len(txt)-1),
		uintptr(unsafe.Pointer(&rc)),
		dtCenter|dtVCenter|dtSingleLine|dtNoClip,
	)
}

// fillRect fills the given rectangle with a solid colour.
func fillRect(hdc HDC, x, y, w, h int, clr uintptr) {
	br, _, _ := pCreateSolidBrush.Call(clr)
	defer pDeleteObject.Call(br)
	rc := rect{int32(x), int32(y), int32(x + w), int32(y + h)}
	pFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), uintptr(br))
}

// beginDoubleBuffer returns an off-screen HDC for flicker-free painting.
// The returned flush function copies the result to screen and frees resources.
func beginDoubleBuffer(hdc HDC, w, h int) (mem HDC, bmp HBITMAP, flush func()) {
	m, _, _ := pCreateCompatibleDC.Call(uintptr(hdc))
	b, _, _ := pCreateCompatibleBitmap.Call(uintptr(hdc), uintptr(w), uintptr(h))
	pSelectObject.Call(m, b)
	flush = func() {
		pBitBlt.Call(uintptr(hdc), 0, 0, uintptr(w), uintptr(h), m, 0, 0, srcCopy)
		pDeleteObject.Call(uintptr(b))
		pDeleteDC.Call(m)
	}
	return HDC(m), HBITMAP(b), flush
}
