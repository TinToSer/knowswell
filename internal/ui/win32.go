// Win32 type aliases, constants and proc bindings.
//
// All values here come from the Windows headers (winuser.h, wingdi.h,
// etc.). They are declared as plain Go types so the rest of the package
// can be written without unsafe.Pointer casts scattered throughout.
package ui

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------- Handle type aliases ----------
//
// All Win32 handle types are pointer-sized integers on all supported
// Windows architectures. We define them as uintptr so that values
// returned by .Call() (which always returns uintptr) can be stored
// directly without casts. HWND comes from golang.org/x/sys/windows
// which already defines it as syscall.Handle = uintptr on Windows.

type (
	HWND      = windows.HWND // syscall.Handle = uintptr on Windows
	HDC       = uintptr
	HINSTANCE = uintptr
	HBRUSH    = uintptr
	HFONT     = uintptr
	HBITMAP   = uintptr
	HPEN      = uintptr
	HCURSOR   = uintptr
	HMENU     = uintptr
)

// Win is a convenience alias used throughout this package.
type Win = HWND

// ---------- Lazy DLL / proc loaders ----------

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	msimg32  = windows.NewLazySystemDLL("msimg32.dll")
	comdlg32 = windows.NewLazySystemDLL("comdlg32.dll")
	comctl32 = windows.NewLazySystemDLL("comctl32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	msftedit = windows.NewLazySystemDLL("msftedit.dll")

	// user32
	pCreateWindowExW            = user32.NewProc("CreateWindowExW")
	pDefWindowProcW             = user32.NewProc("DefWindowProcW")
	pDestroyWindow              = user32.NewProc("DestroyWindow")
	pDispatchMessageW           = user32.NewProc("DispatchMessageW")
	pGetClientRect              = user32.NewProc("GetClientRect")
	pGetDC                      = user32.NewProc("GetDC")
	pGetMessageW                = user32.NewProc("GetMessageW")
	pGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	pGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	pGetWindowRect              = user32.NewProc("GetWindowRect")
	pInvalidateRect             = user32.NewProc("InvalidateRect")
	pLoadCursorW                = user32.NewProc("LoadCursorW")
	pMoveWindow                 = user32.NewProc("MoveWindow")
	pPostQuitMessage            = user32.NewProc("PostQuitMessage")
	pRegisterClassExW           = user32.NewProc("RegisterClassExW")
	pReleaseDC                  = user32.NewProc("ReleaseDC")
	pSendMessageW               = user32.NewProc("SendMessageW")
	pPostMessageW               = user32.NewProc("PostMessageW")
	pSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	pSetTimer                   = user32.NewProc("SetTimer")
	pKillTimer                  = user32.NewProc("KillTimer")
	pSetWindowPos               = user32.NewProc("SetWindowPos")
	pSetWindowTextW             = user32.NewProc("SetWindowTextW")
	pGetWindowTextW             = user32.NewProc("GetWindowTextW")
	pGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	pShowWindow                 = user32.NewProc("ShowWindow")
	pTranslateMessage           = user32.NewProc("TranslateMessage")
	pUpdateWindow               = user32.NewProc("UpdateWindow")
	pBeginPaint                 = user32.NewProc("BeginPaint")
	pEndPaint                   = user32.NewProc("EndPaint")
	pFillRect                   = user32.NewProc("FillRect")
	pDrawTextW                  = user32.NewProc("DrawTextW")
	pGetCursorPos               = user32.NewProc("GetCursorPos")
	pSetCursor                  = user32.NewProc("SetCursor")
	pGetWindowLongW             = user32.NewProc("GetWindowLongW")
	pSetWindowLongW             = user32.NewProc("SetWindowLongW")
	pSetWindowDisplayAffinity   = user32.NewProc("SetWindowDisplayAffinity")
	pSetCapture                 = user32.NewProc("SetCapture")
	pReleaseCapture             = user32.NewProc("ReleaseCapture")
	pScreenToClient             = user32.NewProc("ScreenToClient")
	pClientToScreen             = user32.NewProc("ClientToScreen")
	pGetAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	pTrackMouseEvent            = user32.NewProc("TrackMouseEvent")
	pSetFocus                   = user32.NewProc("SetFocus")

	// clipboard (user32 + kernel32)
	pOpenClipboard    = user32.NewProc("OpenClipboard")
	pCloseClipboard   = user32.NewProc("CloseClipboard")
	pEmptyClipboard   = user32.NewProc("EmptyClipboard")
	pSetClipboardData = user32.NewProc("SetClipboardData")
	pGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	pGlobalLock       = kernel32.NewProc("GlobalLock")
	pGlobalUnlock     = kernel32.NewProc("GlobalUnlock")

	// gdi32
	pCreateFontIndirectW    = gdi32.NewProc("CreateFontIndirectW")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pStretchBlt             = gdi32.NewProc("StretchBlt")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pRectangle              = gdi32.NewProc("Rectangle")
	pMoveToEx               = gdi32.NewProc("MoveToEx")
	pLineTo                 = gdi32.NewProc("LineTo")
	pCreateDIBSection       = gdi32.NewProc("CreateDIBSection")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pSetBkColor             = gdi32.NewProc("SetBkColor")
	pSetBkMode              = gdi32.NewProc("SetBkMode")

	// dwmapi
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")

	// comctl32
	pInitCommonControlsEx = comctl32.NewProc("InitCommonControlsEx")

	// comdlg32
	pGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")

	// msimg32
	pAlphaBlend = msimg32.NewProc("AlphaBlend")

	// shell32
	pShellExecuteW = shell32.NewProc("ShellExecuteW")
)

// ---------- Native structs ----------

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         uintptr
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type msg struct {
	Hwnd   HWND
	Msg    uint32
	WParam uintptr
	LParam uintptr
	Time   uint32
	Pt     point
}

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }

type paintStruct struct {
	Hdc         HDC
	FErase      int32
	RcPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type logFont struct {
	LfHeight         int32
	LfWidth          int32
	LfEscapement     int32
	LfOrientation    int32
	LfWeight         int32
	LfItalic         byte
	LfUnderline      byte
	LfStrikeOut      byte
	LfCharSet        byte
	LfOutPrecision   byte
	LfClipPrecision  byte
	LfQuality        byte
	LfPitchAndFamily byte
	LfFaceName       [32]uint16
}

type bitmapInfoHeader struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

type trackMouseEvent struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   HWND
	DwHoverTime uint32
}

type windowPos struct {
	Hwnd            HWND
	HwndInsertAfter HWND
	X               int32
	Y               int32
	Cx              int32
	Cy              int32
	Flags           uint32
}

type openFilename struct {
	LStructSize       uint32
	HwndOwner         HWND
	HInstance         HINSTANCE
	LpstrFilter       *uint16
	LpstrCustomFilter *uint16
	NMaxCustFilter    uint32
	NFilterIndex      uint32
	LpstrFile         *uint16
	NMaxFile          uint32
	LpstrFileTitle    *uint16
	NMaxFileTitle     uint32
	LpstrInitialDir   *uint16
	LpstrTitle        *uint16
	Flags             uint32
	NFileOffset       uint16
	NFileExtension    uint16
	LpstrDefExt       *uint16
	LCustData         uintptr
	LpfnHook          uintptr
	LpTemplateName    *uint16
	PvReserved        unsafe.Pointer
	DwReserved        uint32
	FlagsEx           uint32
}

// editStream is the EDITSTREAM struct used with EM_STREAMIN / EM_STREAMOUT.
// Layout must match the C struct on 64-bit Windows.
type editStream struct {
	DwCookie    uintptr // application-defined value passed to callback
	DwError     uint32  // error code set by callback
	_           [4]byte // padding to align PfnCallback on 64-bit
	PfnCallback uintptr // EDITSTREAMCALLBACK function pointer
}

type initCommonControlsEx struct {
	DwSize uint32
	DwICC  uint32
}

// ---------- Window styles ----------

const (
	wsPopup      = 0x80000000
	wsChild      = 0x40000000
	wsVisible    = 0x10000000
	wsBorder     = 0x00800000
	wsCaption    = 0x00C00000
	wsVScroll    = 0x00200000
	wsThickFrame = 0x00040000
	wsSysMenu    = 0x00080000

	wsExTopMost    = 0x00000008
	wsExLayered    = 0x00080000
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000
	wsExAppWindow  = 0x00040000

	lwaAlpha    = 0x00000002
	lwaColorKey = 0x00000001

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080

	hwndTopMost   = ^uintptr(0) // -1
	hwndNoTopMost = uintptr(0xFFFFFFFE)

	swShow    = 5
	swHide    = 0
	swRestore = 9
)

// ---------- Messages ----------

const (
	wmSetText     = 0x000C
	wmDestroy     = 0x0002
	wmMove        = 0x0003
	wmSize        = 0x0005
	wmPaint       = 0x000F
	wmClose       = 0x0010
	wmEraseBkgnd  = 0x0014
	wmCommand     = 0x0111
	wmTimer       = 0x0113
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmMouseWheel  = 0x020A
	wmKeyDown     = 0x0100
	wmChar        = 0x0102
	wmNCHitTest      = 0x0084
	wmNCLButtonDown  = 0x00A1
	wmMouseLeave     = 0x02A3
	wmApp         = 0x8000
	wmSetCursor          = 0x0020
	wmWindowPosChanging  = 0x0046

	wmCtlColorEdit   = 0x0133
	wmCtlColorStatic = 0x0138

	enChange = 0x0300

	htCaption = 2
	htClient  = 1

	vkReturn  = 0x0D
	vkEscape  = 0x1B
	vkBack    = 0x08
	vkShift   = 0x10
	vkControl = 0x11

	idcArrow = 32512
	idcCross = 32515
	idcIBeam = 32513

	transparent = 1

	gwlExStyle = -20
	gwlStyle   = -16

	psSolid   = 0
	dcBrush   = 18
	dcPen     = 19
	nullBrush = 5

	dtTop        = 0x00000000
	dtLeft       = 0x00000000
	dtCenter     = 0x00000001
	dtRight      = 0x00000002
	dtVCenter    = 0x00000004
	dtWordBreak  = 0x00000010
	dtSingleLine = 0x00000020
	dtNoClip     = 0x00000100

	esLeft        = 0x0000
	esMultiLine   = 0x0004
	esAutoVScroll = 0x0040
	esAutoHScroll = 0x0080
	esWantReturn  = 0x1000

	emSetSel      = 0x00B1
	emReplaceSel  = 0x00C2
	emSetReadOnly = 0x00CF

	wdaNone               = 0x00000000
	wdaMonitor            = 0x00000001
	wdaExcludeFromCapture = 0x00000011

	biRGB        = 0
	dibRGBColors = 0
	srcCopy      = 0x00CC0020

	smCXScreen     = 0
	smCYScreen     = 1
	smXVirtScreen  = 76
	smYVirtScreen  = 77
	smCXVirtScreen = 78
	smCYVirtScreen = 79

	tmeLeave = 0x00000002

	ofnFileMustExist = 0x00001000
	ofnExplorer      = 0x00080000

	cfDIB         = 8
	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	// Combobox styles and messages.
	cbsDropdown    = 0x0002
	cbsHasStrings  = 0x0200
	cbAddString    = 0x0143
	cbResetContent = 0x014B
	cbGetCurSel    = 0x0147
	cbSetCurSel    = 0x014E
	cbnSelChange   = 0x0001

	// Trackbar (slider) control.
	tbsAutoTicks = 0x0001
	tbmSetRange  = 0x0406
	tbmSetPos    = 0x0405
	tbmGetPos    = 0x0400
	wmHScroll    = 0x0114

	// RichEdit control messages.
	wmGetText        = 0x000D
	wmGetTextLength  = 0x000E
	esReadOnly       = 0x0800
	emSetBkgndColor  = 0x0443
	emSetTextEx      = 0x0461 // sets RichEdit content; detects RTF when Codepage=CP_ACP
)

// Custom window messages (start at WM_APP + 1).
const (
	msgLLMChunk = wmApp + 1
	msgLLMDone  = wmApp + 2
	msgLLMError = wmApp + 3
	msgSnipDone = wmApp + 4
)

// ---------- Small utility wrappers ----------

// utf16Ptr converts a Go string to a UTF-16 pointer suitable for Win32
// APIs that take LPCWSTR / LPWSTR parameters.
func utf16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// loWord / hiWord extract the low / high 16 bits of a uintptr (as packed
// into wParam / lParam by Win32 message dispatch).
func loWord(l uintptr) int32 { return int32(int16(l & 0xffff)) }
func hiWord(l uintptr) int32 { return int32(int16((l >> 16) & 0xffff)) }

// makeLong packs two int16 values into a single uintptr (for wParam).
func makeLong(lo, hi int32) uintptr {
	return uintptr(uint16(lo)) | uintptr(uint16(hi))<<16
}

// hInstance returns the module handle of the running executable.
func hInstance() HINSTANCE {
	h, _, _ := pGetModuleHandleW.Call(0)
	return HINSTANCE(h)
}

// rgb packs an 8-bit RGB triple into a Win32 COLORREF.
func rgb(r, g, b byte) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}
