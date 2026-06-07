// clipboard.go — copy an image.Image to the Windows clipboard as CF_DIB.
package ui

import (
	"image"
	"unsafe"
)

// copyImageToClipboard places img onto the clipboard as a 24-bit DIB so
// any application can paste the captured screenshot. Called on the UI
// thread immediately after a snip capture succeeds.
func copyImageToClipboard(img image.Image) {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		return
	}
	bounds := rgba.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	// 24-bit BGR rows, each padded to a 4-byte boundary.
	stride := (w*3 + 3) &^ 3
	pixSize := stride * h
	totalSize := 40 + pixSize // BITMAPINFOHEADER (40 bytes) + pixels

	hMem, _, _ := pGlobalAlloc.Call(gmemMoveable, uintptr(totalSize))
	if hMem == 0 {
		return
	}
	ptr, _, _ := pGlobalLock.Call(hMem)
	if ptr == 0 {
		return
	}

	// Write BITMAPINFOHEADER.
	hdr := (*bitmapInfoHeader)(unsafe.Pointer(ptr))
	hdr.BiSize = 40
	hdr.BiWidth = int32(w)
	hdr.BiHeight = int32(h) // positive → bottom-up DIB
	hdr.BiPlanes = 1
	hdr.BiBitCount = 24
	hdr.BiCompression = biRGB
	hdr.BiSizeImage = uint32(pixSize)

	// Write pixel rows bottom-up in BGR order.
	pix := (*[1 << 28]byte)(unsafe.Pointer(ptr + 40))[:pixSize:pixSize]
	for row := 0; row < h; row++ {
		srcRow := rgba.Pix[(h-1-row)*rgba.Stride : (h-1-row)*rgba.Stride+w*4]
		dstOff := row * stride
		for col := 0; col < w; col++ {
			s := col * 4
			d := dstOff + col*3
			pix[d+0] = srcRow[s+2] // B
			pix[d+1] = srcRow[s+1] // G
			pix[d+2] = srcRow[s+0] // R
		}
	}

	pGlobalUnlock.Call(hMem)

	pOpenClipboard.Call(0)
	pEmptyClipboard.Call()
	pSetClipboardData.Call(cfDIB, hMem)
	pCloseClipboard.Call()
	// Clipboard now owns hMem; do not GlobalFree it.
}
