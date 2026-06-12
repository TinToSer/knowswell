// Package capture implements Win32 GDI screen capture.
//
// It uses GetDC + BitBlt to copy pixels from the screen (or a region
// thereof) into a DIB section, then converts the raw BGRA buffer to
// a Go image.Image.
package capture

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Lazy-loaded DLL handles. Using NewLazySystemDLL defers LoadLibrary
// until the first call to .NewProc / .Handle, which keeps startup fast
// and allows the package to compile cleanly even if DLLs are missing.
var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procGetDC            = user32.NewProc("GetDC")
	procReleaseDC        = user32.NewProc("ReleaseDC")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")

	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
)

const (
	srccopy        = 0x00CC0020
	smCXScreen     = 0
	smCYScreen     = 1
	smXVirtScreen  = 76
	smYVirtScreen  = 77
	smCXVirtScreen = 78
	smCYVirtScreen = 79
	biRGB          = 0
	dibRGBColors   = 0
)

// bitmapInfoHeader is the Win32 BITMAPINFOHEADER structure.
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

// bitmapInfo is a minimal BITMAPINFO (header + 1 colour table entry).
type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

// CaptureRegion captures the rectangular region (x, y, w, h) of the
// primary display and returns it as an *image.RGBA.
//
// The rectangle is clamped to the virtual screen bounds, so calling with
// coordinates that extend beyond a monitor will not crash (pixels outside
// the display are returned as opaque black).
func CaptureRegion(x, y, w, h int) (image.Image, error) {
	if w <= 0 || h <= 0 {
		return nil, os.ErrInvalid
	}

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, os.ErrInvalid
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, os.ErrInvalid
	}
	defer procDeleteDC.Call(hdcMem)

	hBmp, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(w), uintptr(h))
	if hBmp == 0 {
		return nil, os.ErrInvalid
	}
	defer procDeleteObject.Call(hBmp)

	oldBmp, _, _ := procSelectObject.Call(hdcMem, hBmp)
	defer procSelectObject.Call(hdcMem, oldBmp)

	procBitBlt.Call(
		hdcMem, 0, 0, uintptr(w), uintptr(h),
		hdcScreen, uintptr(x), uintptr(y),
		srccopy,
	)

	bi := bitmapInfo{}
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(w)
	bi.BmiHeader.BiHeight = -int32(h) // top-down
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = biRGB

	pixels := make([]byte, w*h*4)
	procGetDIBits.Call(
		hdcMem, hBmp, 0, uintptr(h),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
	)

	// DIB bits are in BGRA byte order. Convert to RGBA for image.RGBA.
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	stride := w * 4
	for py := 0; py < h; py++ {
		row := img.Pix[py*img.Stride : py*img.Stride+w*4]
		for px := 0; px < w; px++ {
			i := px * 4
			row[i+0] = pixels[py*stride+i+2] // R <- B
			row[i+1] = pixels[py*stride+i+1] // G
			row[i+2] = pixels[py*stride+i+0] // B <- R
			row[i+3] = 255                   // A
		}
	}
	return img, nil
}

// SavePNG writes img to path as a PNG file. Any error from the
// underlying os.Create / png.Encode is returned to the caller.
func SavePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ScreenSize returns the width and height of the primary monitor in pixels.
func ScreenSize() (int, int) {
	w, _, _ := procGetSystemMetrics.Call(smCXScreen)
	h, _, _ := procGetSystemMetrics.Call(smCYScreen)
	return int(w), int(h)
}

// VirtualScreenBounds returns the bounding rectangle of the virtual
// desktop (all monitors combined) in screen coordinates.
func VirtualScreenBounds() (x, y, w, h int) {
	vx, _, _ := procGetSystemMetrics.Call(smXVirtScreen)
	vy, _, _ := procGetSystemMetrics.Call(smYVirtScreen)
	vw, _, _ := procGetSystemMetrics.Call(smCXVirtScreen)
	vh, _, _ := procGetSystemMetrics.Call(smCYVirtScreen)
	return int(int32(vx)), int(int32(vy)), int(vw), int(vh)
}

// Color is a re-export so callers don't need to import image/color.
type Color = color.RGBA
