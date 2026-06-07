//go:build windows

package ui

import (
	"bytes"
	"image"
	"image/png"
	"sync"
	"sync/atomic"
	"unsafe"

	"knowswell/assets"
)

var (
	logoOnce    sync.Once
	logoDC      HDC
	logoNativeW int
	logoNativeH int
	logoReady   atomic.Int32
)

func loadLogo() {
	logoOnce.Do(func() {
		img := openCompanyPNG()
		if img == nil {
			return
		}
		bounds := img.Bounds()
		logoNativeW = bounds.Dx()
		logoNativeH = bounds.Dy()
		dc, _, _, _ := imageToDC(img)
		logoDC = dc
		logoReady.Store(1)
	})
}

func logoLoaded() bool { return logoReady.Load() != 0 }

func openCompanyPNG() image.Image {
	if len(assets.LogoPNG) == 0 {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(assets.LogoPNG))
	if err != nil {
		return nil
	}
	return img
}

func imageToDC(img image.Image) (HDC, HBITMAP, int, int) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0
	}

	screenDC, _, _ := pGetDC.Call(0)
	defer pReleaseDC.Call(0, screenDC)

	dc, _, _ := pCreateCompatibleDC.Call(screenDC)

	bi := bitmapInfo{}
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(w)
	bi.BmiHeader.BiHeight = -int32(h)
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = biRGB

	var pBits unsafe.Pointer
	hBmp, _, _ := pCreateDIBSection.Call(
		screenDC,
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&pBits)),
		0, 0,
	)
	if hBmp == 0 {
		pDeleteDC.Call(dc)
		return 0, 0, 0, 0
	}
	pSelectObject.Call(dc, hBmp)

	if pBits != nil {
		pixels := (*[1 << 28]byte)(pBits)[:w*h*4 : w*h*4]
		for py := 0; py < h; py++ {
			for px := 0; px < w; px++ {
				c := img.At(px+bounds.Min.X, py+bounds.Min.Y)
				r16, g16, b16, a16 := c.RGBA()
				a := byte(a16 >> 8)
				r := byte(uint32(r16>>8) * uint32(a) / 255)
				g := byte(uint32(g16>>8) * uint32(a) / 255)
				b := byte(uint32(b16>>8) * uint32(a) / 255)
				i := (py*w + px) * 4
				pixels[i+0] = b
				pixels[i+1] = g
				pixels[i+2] = r
				pixels[i+3] = a
			}
		}
	}
	return HDC(dc), HBITMAP(hBmp), w, h
}

func drawLogo(hdc HDC, x, y, w, h int) {
	if !logoLoaded() || logoDC == 0 || logoNativeW <= 0 || logoNativeH <= 0 {
		return
	}
	// AC_SRC_OVER | alpha=255 | AC_SRC_ALPHA
	bf := uint32(0) | uint32(0)<<8 | uint32(255)<<16 | uint32(1)<<24
	pAlphaBlend.Call(
		uintptr(hdc),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(logoDC),
		0, 0, uintptr(logoNativeW), uintptr(logoNativeH),
		uintptr(bf),
	)
}
