// Settings window: API credentials, system-prompt presets, ghost mode, quit.
package ui

import (
	"syscall"
	"unsafe"

	"knowswell/internal/config"
)

// Settings window dimensions and layout constants.
const (
	swW   = 420
	swPad = 14
	swHeaderH     = 36
	swEditH       = 26
	swLabelH      = 14
	swPresetSaveW = 96

	// y-positions (label then control).
	swYAPIKeyLbl = swHeaderH + 6
	swYAPIKey    = swYAPIKeyLbl + swLabelH + 2
	swYModelLbl  = swYAPIKey + swEditH + 8
	swYModel     = swYModelLbl + swLabelH + 2
	swYEPLbl     = swYModel + swEditH + 8
	swYEP        = swYEPLbl + swLabelH + 2
	swYPresetLbl = swYEP + swEditH + 12
	swYPreset    = swYPresetLbl + swLabelH + 2
	swYPromptLbl = swYPreset + swEditH + 8
	swYPrompt    = swYPromptLbl + swLabelH + 2
	swPromptH     = 84
	swYGhost      = swYPrompt + swPromptH + 14
	swYOpacityLbl = swYGhost + 30
	swYOpacity    = swYOpacityLbl + swLabelH + 2
	swH           = swYOpacity + 30 + 14 + 34 + 14 // slider + gap + btn + bottom pad

	// combobox width (leaves room for Save Preset button)
	swPresetComboW = swW - swPad*2 - swPresetSaveW - 6

	// control ID for the preset combobox (routed via WM_COMMAND)
	ctrlPresetCombo = 101
)

func settingsWndProcImpl(hwnd, umsg, wParam, lParam uintptr) (result uintptr) {
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
		paintSettings(hwnd, a)
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
		if x >= w-66 && x <= w-14 && y >= 5 && y <= 29 {
			return htClient
		}
		if y < swHeaderH {
			return htCaption
		}
		return htClient
	case wmLButtonDown:
		x, y := loWord(lParam), hiWord(lParam)
		a.settingsClick(hwnd, int(x), int(y))
		return 0
	case wmCommand:
		ctrlID := int(wParam & 0xFFFF)
		notif := int((wParam >> 16) & 0xFFFF)
		if ctrlID == ctrlPresetCombo && notif == cbnSelChange {
			a.settingsPresetSelected()
		}
		return 0
	case wmCtlColorEdit:
		hdc := HDC(wParam)
		pSetTextColor.Call(uintptr(hdc), clrInputText)
		pSetBkColor.Call(uintptr(hdc), clrSurface2)
		br, _, _ := pCreateSolidBrush.Call(clrSurface2)
		return uintptr(br)
	case wmHScroll:
		if HWND(lParam) == a.hwOpacity {
			pos, _, _ := pSendMessageW.Call(uintptr(a.hwOpacity), tbmGetPos, 0, 0)
			a.settingsOpacity = int(pos)
			a.cfg.Opacity = a.settingsOpacity
			a.applyOpacity()
			invalidate(HWND(hwnd))
		}
		return 0
	case wmDestroy:
		a.Settings = 0
		a.hwAPIKey = 0
		a.hwModel = 0
		a.hwEndpoint = 0
		a.hwSysPrompt = 0
		a.hwPresets = 0
		a.hwOpacity = 0
		return 0
	}
	return defProc(hwnd, umsg, wParam, lParam)
}

func showSettings(a *App) {
	if a.Settings != 0 {
		setTopmost(a.Settings)
		pShowWindow.Call(uintptr(a.Settings), swShow)
		return
	}

	sw, sh := screenSize()
	sX, sY := (sw-swW)/2, (sh-swH)/2

	h, _, _ := pCreateWindowExW.Call(
		wsExTopMost|wsExLayered|wsExToolWindow,
		uintptr(unsafe.Pointer(utf16Ptr("SASettings"))),
		0,
		wsPopup|wsVisible,
		uintptr(sX), uintptr(sY), uintptr(swW), uintptr(swH),
		0, 0, uintptr(hInstance()), 0,
	)
	a.Settings = HWND(h)
	noRound := uint32(1)
	pDwmSetWindowAttribute.Call(uintptr(h), 33, uintptr(unsafe.Pointer(&noRound)), 4)
	pSetWindowDisplayAffinity.Call(uintptr(h), a.displayAffinity())
	pSetLayeredWindowAttributes.Call(uintptr(h), 0, uintptr(a.alphaValue()), lwaAlpha)

	a.settingsGhostMode = a.cfg.GhostMode
	a.settingsOpacity = a.cfg.Opacity

	pad := swPad
	editW := swW - pad*2

	a.hwAPIKey = createEditControl(HWND(h), a.cfg.APIKey, pad, swYAPIKey, editW, swEditH)
	a.hwModel = createEditControl(HWND(h), a.cfg.Model, pad, swYModel, editW, swEditH)
	a.hwEndpoint = createEditControl(HWND(h), a.cfg.APIEndpoint, pad, swYEP, editW, swEditH)

	// Preset combobox.
	hc, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("COMBOBOX"))),
		0,
		wsChild|wsVisible|cbsDropdown|cbsHasStrings,
		uintptr(pad), uintptr(swYPreset),
		uintptr(swPresetComboW), 200,
		uintptr(h), uintptr(ctrlPresetCombo), uintptr(hInstance()), 0,
	)
	a.hwPresets = HWND(hc)
	pSendMessageW.Call(uintptr(hc), wmSetFont, uintptr(getFont("Courier New", 11, false)), 1)
	populatePresetCombo(a)

	// System prompt textarea (multiline).
	a.hwSysPrompt = createMultilineEdit(HWND(h), a.cfg.SystemPrompt,
		pad, swYPrompt, editW, swPromptH)

	// Opacity trackbar.
	htb, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("msctls_trackbar32"))),
		0,
		wsChild|wsVisible|tbsAutoTicks,
		uintptr(pad), uintptr(swYOpacity),
		uintptr(editW-50), 26,
		uintptr(h), 0, uintptr(hInstance()), 0,
	)
	a.hwOpacity = HWND(htb)
	pSendMessageW.Call(uintptr(htb), tbmSetRange, 1, makeLong(10, 100))
	pSendMessageW.Call(uintptr(htb), tbmSetPos, 1, uintptr(a.cfg.Opacity))

	pShowWindow.Call(uintptr(h), swShow)
	pUpdateWindow.Call(uintptr(h))
	setTopmost(HWND(h))
}

func createMultilineEdit(parent HWND, text string, x, y, w, h int) HWND {
	eh, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		wsChild|wsVisible|wsBorder|esMultiLine|esAutoVScroll|esWantReturn,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(parent), 0, uintptr(hInstance()), 0,
	)
	pSendMessageW.Call(uintptr(eh), wmSetFont, uintptr(getFont("Courier New", 11, false)), 1)
	return HWND(eh)
}

func populatePresetCombo(a *App) {
	if a.hwPresets == 0 {
		return
	}
	pSendMessageW.Call(uintptr(a.hwPresets), cbResetContent, 0, 0)
	for _, p := range a.cfg.Prompts {
		str := utf16Ptr(p.Name)
		pSendMessageW.Call(uintptr(a.hwPresets), cbAddString, 0, uintptr(unsafe.Pointer(str)))
	}
	for i, p := range a.cfg.Prompts {
		if p.Text == a.cfg.SystemPrompt {
			pSendMessageW.Call(uintptr(a.hwPresets), cbSetCurSel, uintptr(i), 0)
			break
		}
	}
}

func paintSettings(hwnd uintptr, a *App) {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)

	memDC, _, flush := beginDoubleBuffer(hdc, w, h)
	defer flush()

	drawRoundRect(memDC, 0, 0, w, h, 2, clrBackground, clrBorder)
	drawRoundRect(memDC, 0, 0, w, swHeaderH, 2, clrSurface, 0)
	drawText(memDC, "⚙  SETTINGS", swPad, 7, w-80, 22, clrAccent, 12, true)
	drawRoundRect(memDC, w-66, 5, 52, 24, 2, clrDanger, 0)
	drawCenteredText(memDC, "[CLOSE]", w-66, 5, 52, 24, rgb(0, 0, 0), 9, true)

	pad := swPad
	drawText(memDC, "API KEY", pad, swYAPIKeyLbl, w-pad*2, swLabelH, clrTextDim, 11, true)
	drawText(memDC, "MODEL", pad, swYModelLbl, w-pad*2, swLabelH, clrTextDim, 11, true)
	drawText(memDC, "API ENDPOINT", pad, swYEPLbl, w-pad*2, swLabelH, clrTextDim, 11, true)

	// Preset row label + Save Preset button (combobox is a native child, drawn by Windows).
	drawText(memDC, "PRESET", pad, swYPresetLbl, 60, swLabelH, clrTextDim, 11, true)
	savePresetX := w - pad - swPresetSaveW
	drawRoundRect(memDC, savePresetX, swYPreset-1, swPresetSaveW, swEditH+2, 2, clrAccent, 0)
	drawCenteredText(memDC, "[SAVE]", savePresetX, swYPreset-1, swPresetSaveW, swEditH+2, rgb(0, 25, 8), 10, true)

	drawText(memDC, "SYSTEM PROMPT", pad, swYPromptLbl, w-pad*2, swLabelH, clrTextDim, 11, true)

	// Ghost Mode toggle pill.
	ghostX, ghostY := pad, swYGhost
	tW, tH := 44, 22
	if a.settingsGhostMode {
		drawRoundRect(memDC, ghostX, ghostY, tW, tH, tH/2, clrAccent, 0)
		drawRoundRect(memDC, ghostX+tW-tH+2, ghostY+2, tH-4, tH-4, (tH-4)/2, rgb(255, 255, 255), 0)
	} else {
		drawRoundRect(memDC, ghostX, ghostY, tW, tH, tH/2, clrSurface2, clrBorder)
		drawRoundRect(memDC, ghostX+2, ghostY+2, tH-4, tH-4, (tH-4)/2, clrTextDim, 0)
	}
	drawText(memDC, "GHOST MODE  --  exclude windows from screen capture",
		ghostX+tW+8, ghostY+4, w-pad-ghostX-tW-8, 16, clrText, 11, true)

	// Opacity label + current value (trackbar is drawn by Windows).
	opLabel := "OPACITY: " + itoa(a.settingsOpacity) + "%"
	drawText(memDC, opLabel, pad, swYOpacityLbl, w-pad*2, swLabelH, clrTextDim, 11, true)

	// Bottom buttons.
	btnsY := h - pad - 34
	drawRoundRect(memDC, pad, btnsY, 160, 34, 2, clrAccent, 0)
	drawCenteredText(memDC, "[SAVE SETTINGS]", pad, btnsY, 160, 34, rgb(0, 25, 8), 11, true)
	drawRoundRect(memDC, w-pad-90, btnsY, 90, 34, 2, clrDanger, 0)
	drawCenteredText(memDC, "[QUIT]", w-pad-90, btnsY, 90, 34, rgb(25, 0, 0), 12, true)
}

func (a *App) settingsClick(hwnd uintptr, x, y int) {
	var rc rect
	pGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rc)))
	w, h := int(rc.Right), int(rc.Bottom)

	// Close.
	if x >= w-66 && x <= w-14 && y >= 5 && y <= 29 {
		pDestroyWindow.Call(uintptr(hwnd))
		return
	}
	// Save Preset.
	savePresetX := w - swPad - swPresetSaveW
	if x >= savePresetX && x <= w-swPad && y >= swYPreset-1 && y <= swYPreset+swEditH+1 {
		a.settingsSavePreset()
		return
	}
	// Ghost Mode toggle.
	if x >= swPad && x <= swPad+250 && y >= swYGhost && y <= swYGhost+22 {
		a.settingsGhostMode = !a.settingsGhostMode
		invalidate(HWND(hwnd))
		return
	}
	// Save Settings.
	btnsY := h - swPad - 34
	if x >= swPad && x <= swPad+160 && y >= btnsY && y <= btnsY+34 {
		a.saveSettings()
		pDestroyWindow.Call(uintptr(hwnd))
		return
	}
	// Quit.
	if x >= w-swPad-90 && x <= w-swPad && y >= btnsY && y <= btnsY+34 {
		pDestroyWindow.Call(uintptr(hwnd))
		pPostQuitMessage.Call(0)
		return
	}
}

func (a *App) settingsSavePreset() {
	if a.hwPresets == 0 || a.hwSysPrompt == 0 {
		return
	}
	name := editControlText(a.hwPresets)
	text := editControlText(a.hwSysPrompt)
	if name == "" || text == "" {
		return
	}
	for i, p := range a.cfg.Prompts {
		if p.Name == name {
			a.cfg.Prompts[i].Text = text
			_ = a.cfg.Save()
			return
		}
	}
	a.cfg.Prompts = append(a.cfg.Prompts, config.PromptPreset{Name: name, Text: text})
	_ = a.cfg.Save()
	populatePresetCombo(a)
}

func (a *App) settingsPresetSelected() {
	if a.hwPresets == 0 || a.hwSysPrompt == 0 {
		return
	}
	idx, _, _ := pSendMessageW.Call(uintptr(a.hwPresets), cbGetCurSel, 0, 0)
	if int32(idx) < 0 || int(idx) >= len(a.cfg.Prompts) {
		return
	}
	text := a.cfg.Prompts[idx].Text
	pSetWindowTextW.Call(uintptr(a.hwSysPrompt),
		uintptr(unsafe.Pointer(utf16Ptr(text))))
	// Apply instantly to the live session and persist to disk immediately
	// so a restart also uses the selected preset.
	a.llm.SystemPrompt = text
	a.cfg.SystemPrompt = text
	_ = a.cfg.Save()
}

func (a *App) saveSettings() {
	if a.hwAPIKey != 0 {
		a.cfg.APIKey = editControlText(a.hwAPIKey)
	}
	if a.hwModel != 0 {
		a.cfg.Model = editControlText(a.hwModel)
	}
	if a.hwEndpoint != 0 {
		a.cfg.APIEndpoint = editControlText(a.hwEndpoint)
	}
	if a.hwSysPrompt != 0 {
		a.cfg.SystemPrompt = editControlText(a.hwSysPrompt)
	}
	a.cfg.GhostMode = a.settingsGhostMode
	a.cfg.Opacity = a.settingsOpacity
	_ = a.cfg.Save()
	a.llm.APIKey = a.cfg.APIKey
	a.llm.Model = a.cfg.Model
	a.llm.Endpoint = a.cfg.APIEndpoint
	a.llm.SystemPrompt = a.cfg.SystemPrompt
}

func createEditControl(parent HWND, text string, x, y, w, h int) HWND {
	eh, _, _ := pCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(utf16Ptr("EDIT"))),
		uintptr(unsafe.Pointer(utf16Ptr(text))),
		wsChild|wsVisible|wsBorder|esAutoHScroll,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(parent), 0, uintptr(hInstance()), 0,
	)
	pSendMessageW.Call(uintptr(eh), wmSetFont,
		uintptr(getFont("Courier New", 11, false)), 1)
	return HWND(eh)
}

func editControlText(hwnd HWND) string {
	n, _, _ := pGetWindowTextLengthW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	pGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(n+1))
	return syscall.UTF16ToString(buf)
}
