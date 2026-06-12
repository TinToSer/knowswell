// Chat-bar window: attached image thumbnails, a text-input EDIT control,
// and Send/Clear buttons. Submits a streaming LLM request on Send.
package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"knowswell/internal/llm"
)

// Chat bar layout constants.
const (
	cbMinH    = 140
	cbHeaderH = 36
	cbPad     = 10
	attThumb  = 60
	attStripH = 76
	editH     = 64
	btnH      = 32
)

const wmSetFont = 0x0030

// chatBarWndProcImpl is the WndProc for the SAChatBar class.
func chatBarWndProcImpl(hwnd, umsg, wParam, lParam uintptr) (result uintptr) {
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
		paintChatBar(hwnd, a)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmCommand:
		return 0
	case wmLButtonDown:
		x, y := loWord(lParam), hiWord(lParam)
		a.chatBarClick(hwnd, int(x), int(y))
		return 0
	case wmMouseMove:
		x, y := loWord(lParam), hiWord(lParam)
		a.chatBarHover(int(x), int(y))
		return 0
	case wmKeyDown:
		if wParam == vkReturn {
			ctrlDown, _, _ := pGetAsyncKeyState.Call(vkControl)
			if ctrlDown&0x8000 != 0 {
				a.submitAsk()
				return 0
			}
		}
		return 0
	case wmCtlColorEdit:
		hdc := HDC(wParam)
		pSetTextColor.Call(uintptr(hdc), clrInputText)
		pSetBkColor.Call(uintptr(hdc), clrSurface2)
		pSetBkMode.Call(uintptr(hdc), transparent)
		br, _, _ := pCreateSolidBrush.Call(clrSurface2)
		return uintptr(br)
	case msgLLMChunk:
		chunk := getMsgString(uint64(lParam))
		a.AppendStream(chunk)
		showOverlay(a, a.StreamText())
		return 0
	case msgLLMDone:
		showOverlay(a, a.StreamText())
		return 0
	case msgLLMError:
		errMsg := getMsgString(uint64(lParam))
		showOverlay(a, "!! ERROR: "+errMsg)
		return 0
	case wmNCHitTest:
		pt := point{X: int32(int16(loWord(lParam))), Y: int32(int16(hiWord(lParam)))}
		pScreenToClient.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
		x, y := int(pt.X), int(pt.Y)
		var rc rect
		pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
		w := int(rc.Right)
		if x >= w-30 && x <= w-6 && y >= 5 && y <= 29 {
			return htClient
		}
		if y < cbHeaderH {
			return htCaption
		}
		return htClient
	case wmDestroy:
		a.ChatBar = 0
		a.editHandle = 0
		return 0
	}
	return defProc(hwnd, umsg, wParam, lParam)
}

// ---------- Show / hide / layout ----------

func showChatBar(a *App) {
	if a.ChatBar != 0 {
		pShowWindow.Call(uintptr(a.ChatBar), swShow)
		pUpdateWindow.Call(uintptr(a.ChatBar))
		layoutChatBar(a)
		return
	}

	w := tbExpandedW
	x, y := a.tbX, a.tbY+tbHeight+4
	h, _, _ := pCreateWindowExW.Call(
		wsExTopMost|wsExLayered|wsExToolWindow,
		uintptr(unsafe.Pointer(utf16Ptr("SAChatBar"))),
		0,
		wsPopup|wsVisible,
		uintptr(x), uintptr(y),
		uintptr(w), uintptr(cbMinH),
		0, 0, uintptr(hInstance()), 0,
	)
	a.ChatBar = HWND(h)
	noRound := uint32(1)
	pDwmSetWindowAttribute.Call(uintptr(h), 33, uintptr(unsafe.Pointer(&noRound)), 4)
	pSetWindowDisplayAffinity.Call(uintptr(h), a.displayAffinity())
	pSetLayeredWindowAttributes.Call(uintptr(h), 0, uintptr(a.alphaValue()), lwaAlpha)

	eh, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
		0,
		wsChild|wsVisible|esMultiLine|esAutoVScroll|esWantReturn|esLeft,
		uintptr(cbPad), uintptr(cbHeaderH+cbPad),
		uintptr(w-cbPad*2), uintptr(editH),
		uintptr(h), 0, uintptr(hInstance()), 0,
	)
	a.editHandle = HWND(eh)
	pSendMessageW.Call(uintptr(eh), wmSetFont,
		uintptr(getFont("Courier New", 12, false)), 1)

	pShowWindow.Call(uintptr(h), swShow)
	pUpdateWindow.Call(uintptr(h))
	setTopmost(a.ChatBar)
	layoutChatBar(a)
	pSetFocus.Call(uintptr(eh))
}

func hideChatBar(a *App) {
	if a.ChatBar != 0 {
		pShowWindow.Call(uintptr(a.ChatBar), swHide)
	}
}

func (a *App) RefreshChatBar() {
	if a.ChatBar != 0 {
		layoutChatBar(a)
		invalidate(a.ChatBar)
	}
}

func layoutChatBar(a *App) {
	if a.ChatBar == 0 {
		return
	}
	w := tbExpandedW
	atts := a.Attachments()
	totalH := cbHeaderH + cbPad + editH + cbPad + btnH + cbPad
	if len(atts) > 0 {
		totalH += attStripH + cbPad
	}
	pMoveWindow.Call(
		uintptr(a.ChatBar),
		uintptr(a.tbX), uintptr(a.tbY+tbHeight+4),
		uintptr(w), uintptr(totalH), 1,
	)
	editY := cbHeaderH + cbPad
	if len(atts) > 0 {
		editY += attStripH + cbPad
	}
	if a.editHandle != 0 {
		pMoveWindow.Call(
			uintptr(a.editHandle),
			uintptr(cbPad), uintptr(editY),
			uintptr(w-cbPad*2), uintptr(editH), 1,
		)
	}
	a.cbH = totalH
}

// ---------- Painting ----------

func paintChatBar(hwnd uintptr, a *App) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)

	memDC, _, flush := beginDoubleBuffer(hdc, w, h)
	defer flush()

	drawRoundRect(memDC, 0, 0, w, h, 2, clrBackground, clrBorder)

	// Header.
	drawRoundRect(memDC, 0, 0, w, cbHeaderH, 2, clrSurface, 0)
	fillRect(memDC, 0, cbHeaderH/2, w, cbHeaderH/2, clrSurface)
	drawText(memDC, ">> ASK KNOWSWELL", cbPad, 8, w-50, 20, clrAccent, 12, true)
	drawRoundRect(memDC, w-30, 5, 24, 24, 2, clrDanger, 0)
	drawCenteredText(memDC, "[X]", w-30, 5, 24, 24, rgb(0, 0, 0), 9, true)

	atts := a.Attachments()
	y := cbHeaderH + cbPad

	if len(atts) > 0 {
		drawRoundRect(memDC, cbPad, y, w-cbPad*2, attStripH, 2, clrSurface, clrBorder)
		x := cbPad * 2
		for i, att := range atts {
			paintAttachmentThumb(memDC, att, x, y+8, attThumb, attThumb-16, i)
			x += attThumb + 8
		}
		addX := cbPad*2 + len(atts)*(attThumb+8)
		drawRoundRect(memDC, addX, y+8, attThumb, attThumb-16, 2, clrSurface2, clrBorder)
		drawCenteredText(memDC, "[+]", addX, y+8, attThumb, attThumb-16, clrAccent, 11, true)
		y += attStripH + cbPad
	}

	// Edit border.
	drawRoundRect(memDC, cbPad-2, y-2, w-cbPad*2+4, editH+4, 2, 0, clrBorder)
	y += editH + cbPad

	// Buttons.
	sendW, clearW := 90, 80
	sendX := w - sendW - cbPad
	clearX := sendX - clearW - 6
	drawRoundRect(memDC, sendX, y, sendW, btnH, 2, clrAccent, 0)
	drawCenteredText(memDC, "[SEND]", sendX, y, sendW, btnH, rgb(0, 25, 8), 12, true)
	drawRoundRect(memDC, clearX, y, clearW, btnH, 2, clrSurface2, clrBorder)
	drawCenteredText(memDC, "[CLEAR]", clearX, y, clearW, btnH, clrTextDim, 11, true)

	if n := len(atts); n > 0 {
		drawText(memDC, itoa(n)+" att.", cbPad, y+8, 80, 16, clrTextDim, 11, true)
	}

	_ = h
}

func paintAttachmentThumb(hdc HDC, att llm.Attachment, x, y, w, h, idx int) {
	drawRoundRect(hdc, x, y, w, h, 2, clrSurface2, clrBorder)

	if att.Image != nil {
		drawImageThumb(hdc, att, x+2, y+2, w-4, h-20)
	} else {
		drawCenteredText(hdc, "📄", x, y, w, h-20, clrTextDim, 20, false)
	}

	label := att.Label
	if len(label) > 8 {
		label = label[:8] + ".."
	}
	drawCenteredText(hdc, label, x, y+h-16, w, 14, clrTextDim, 9, false)
	drawText(hdc, "×", x+w-12, y, 12, 14, clrDanger, 11, true)
}

func drawImageThumb(hdc HDC, att llm.Attachment, x, y, w, h int) {
	img := att.Image
	bounds := img.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	if sw <= 0 || sh <= 0 {
		return
	}

	hdcImg, _, _ := pCreateCompatibleDC.Call(uintptr(hdc))
	defer pDeleteDC.Call(hdcImg)

	bi := bitmapInfo{}
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(sw)
	bi.BmiHeader.BiHeight = -int32(sh)
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = biRGB

	var pBits unsafe.Pointer
	hBmp, _, _ := pCreateDIBSection.Call(
		hdcImg,
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&pBits)),
		0, 0,
	)
	if hBmp == 0 {
		return
	}
	defer pDeleteObject.Call(hBmp)
	pSelectObject.Call(hdcImg, hBmp)

	if pBits != nil {
		pixels := (*[1 << 28]byte)(pBits)[: sw*sh*4 : sw*sh*4]
		for py := 0; py < sh; py++ {
			for px := 0; px < sw; px++ {
				c := img.At(px+bounds.Min.X, py+bounds.Min.Y)
				r32, g32, b32, _ := c.RGBA()
				i := (py*sw + px) * 4
				pixels[i] = byte(b32 >> 8)
				pixels[i+1] = byte(g32 >> 8)
				pixels[i+2] = byte(r32 >> 8)
				pixels[i+3] = 255
			}
		}
	}

	pStretchBlt.Call(
		uintptr(hdc), uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		hdcImg, 0, 0, uintptr(sw), uintptr(sh),
		srcCopy,
	)
}

// ---------- Click handling ----------

func (a *App) chatBarClick(hwnd uintptr, x, y int) {
	atts := a.Attachments()
	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w := int(rc.Right)

	if x >= w-30 && x <= w-6 && y >= 5 && y <= 29 {
		hideChatBar(a)
		return
	}

	buttonY := a.cbH - btnH - cbPad
	sendW, clearW := 90, 80
	sendX := w - sendW - cbPad
	clearX := sendX - clearW - 6

	if x >= sendX && x <= sendX+sendW && y >= buttonY && y <= buttonY+btnH {
		a.submitAsk()
		return
	}
	if x >= clearX && x <= clearX+clearW && y >= buttonY && y <= buttonY+btnH {
		a.ClearAttachments()
		if a.editHandle != 0 {
			pSetWindowTextW.Call(uintptr(a.editHandle),
				uintptr(unsafe.Pointer(utf16Ptr(""))))
			pSetFocus.Call(uintptr(a.editHandle))
		}
		return
	}

	stripY := cbHeaderH + cbPad + 8
	if len(atts) > 0 {
		ax := cbPad * 2
		for i := range atts {
			if x >= ax+attThumb-12 && x < ax+attThumb &&
				y >= stripY && y < stripY+14 {
				a.RemoveAttachment(i)
				layoutChatBar(a)
				invalidate(HWND(hwnd))
				return
			}
			ax += attThumb + 8
		}
		addX := cbPad*2 + len(atts)*(attThumb+8)
		if x >= addX && x < addX+attThumb &&
			y >= stripY && y < stripY+attThumb-16 {
			openFileDialog(HWND(hwnd), a)
		}
	}
}

func (a *App) chatBarHover(x, y int) {
	_ = x
	_ = y
}

// ---------- File picker ----------

func openFileDialog(hwnd HWND, a *App) {
	buf := make([]uint16, 1024)
	filter := syscall.StringToUTF16("Images\x00*.png;*.jpg;*.jpeg;*.bmp;*.gif\x00All Files\x00*.*\x00\x00")
	ofn := openFilename{
		LStructSize: uint32(unsafe.Sizeof(openFilename{})),
		HwndOwner:   hwnd,
		LpstrFilter: &filter[0],
		LpstrFile:   &buf[0],
		NMaxFile:    uint32(len(buf)),
		Flags:       ofnFileMustExist | ofnExplorer,
	}
	r, _, _ := pGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return
	}
	path := syscall.UTF16ToString(buf)
	if path == "" {
		return
	}
	a.AddAttachment(llm.Attachment{
		FilePath: path,
		Label:    filepath.Base(path),
	})
}

// ---------- Submit & streaming ----------

func (a *App) submitAsk() {
	if a.editHandle == 0 {
		return
	}
	query := a.editText()
	atts := a.Attachments()
	if strings.TrimSpace(query) == "" && len(atts) == 0 {
		return
	}
	a.ClearAttachments() // auto-clear snip/attachments once query is submitted
	a.ResetStream()
	showOverlay(a, "> PROCESSING...")
	a.cancelLLM()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	a.attMu.Lock()
	a.llmCancel = cancel
	a.attMu.Unlock()

	hwnd := a.ChatBar
	client := a.llm

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				id := putMsgString(fmt.Sprintf("internal error: %v", r))
				if pp, _, _ := pPostMessageW.Call(uintptr(hwnd), msgLLMError, 0, uintptr(id)); pp == 0 {
					getMsgString(id)
				}
			}
		}()
		err := client.AskStream(ctx, query, atts, func(chunk string) {
			id := putMsgString(chunk)
			r, _, _ := pPostMessageW.Call(uintptr(hwnd), msgLLMChunk, 0, uintptr(id))
			if r == 0 {
				getMsgString(id) // window gone — release the stored string
			}
		})
		if err != nil && err != context.Canceled {
			id := putMsgString(err.Error())
			r, _, _ := pPostMessageW.Call(uintptr(hwnd), msgLLMError, 0, uintptr(id))
			if r == 0 {
				getMsgString(id)
			}
			return
		}
		pPostMessageW.Call(uintptr(hwnd), msgLLMDone, 0, 0)
	}()

	pSetWindowTextW.Call(uintptr(a.editHandle),
		uintptr(unsafe.Pointer(utf16Ptr(""))))
}

func (a *App) editText() string {
	if a.editHandle == 0 {
		return ""
	}
	n, _, _ := pGetWindowTextLengthW.Call(uintptr(a.editHandle))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	pGetWindowTextW.Call(uintptr(a.editHandle),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(n+1))
	return syscall.UTF16ToString(buf)
}

func (a *App) cancelLLM() {
	a.attMu.Lock()
	defer a.attMu.Unlock()
	if a.llmCancel != nil {
		a.llmCancel()
		a.llmCancel = nil
	}
}
