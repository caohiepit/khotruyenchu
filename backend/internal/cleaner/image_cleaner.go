package cleaner

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"

	_ "golang.org/x/image/webp"
)

// CleanCoverWatermark takes raw image bytes (JPEG, PNG, WebP) and removes watermarks
// such as 'truyenfull.vn', 'TRUYENFULL.VN', or bottom watermark text banners.
// It returns clean JPEG image bytes.
func CleanCoverWatermark(r io.Reader) ([]byte, error) {
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 10 || h <= 10 {
		var buf bytes.Buffer
		_ = jpeg.Encode(&buf, src, &jpeg.Options{Quality: 90})
		return buf.Bytes(), nil
	}

	// Create mutable RGBA canvas
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	// 1. Remove Bottom Watermark (e.g., "truyenfull.vn" at bottom 10-15% of cover)
	// Watermark is typically centered horizontally in y: 86% to 97% of height
	cleanBottomWatermark(dst, w, h)

	// 2. Remove Lower-Middle Watermark (e.g., "TRUYENFULL.VN" banner around 62%-72% of height)
	cleanMidWatermarkIfPresent(dst, w, h)

	var outBuf bytes.Buffer
	err = jpeg.Encode(&outBuf, dst, &jpeg.Options{Quality: 92})
	if err != nil {
		return nil, err
	}

	return outBuf.Bytes(), nil
}

// cleanBottomWatermark detects and inpaints the bottom 'truyenfull.vn' watermark band
func cleanBottomWatermark(dst *image.RGBA, w, h int) {
	yStart := int(float64(h) * 0.88)
	yEnd := int(float64(h) * 0.98)
	if yEnd >= h {
		yEnd = h - 1
	}
	if yStart < 0 || yStart >= yEnd {
		return
	}

	xStart := int(float64(w) * 0.08)
	xEnd := int(float64(w) * 0.92)

	// Check if this region has watermark white/bright pixels
	hasWatermark := false
	whiteCount := 0
	for y := yStart; y <= yEnd; y++ {
		for x := xStart; x <= xEnd; x++ {
			c := dst.RGBAAt(x, y)
			if c.R > 215 && c.G > 215 && c.B > 215 {
				whiteCount++
				if whiteCount > 15 {
					hasWatermark = true
					break
				}
			}
		}
		if hasWatermark {
			break
		}
	}

	if !hasWatermark {
		return
	}

	// Inpaint region using vertical gradient interpolation from boundary colors
	inpaintRegion(dst, xStart, yStart, xEnd, yEnd, h)
}

// cleanMidWatermarkIfPresent checks if there's a watermark banner like 'TRUYENFULL.VN' in the lower-mid area
func cleanMidWatermarkIfPresent(dst *image.RGBA, w, h int) {
	yStart := int(float64(h) * 0.62)
	yEnd := int(float64(h) * 0.72)
	xStart := int(float64(w) * 0.18)
	xEnd := int(float64(w) * 0.82)

	// Count high-brightness pixels clustered in this band
	whiteCount := 0
	for y := yStart; y <= yEnd; y++ {
		for x := xStart; x <= xEnd; x++ {
			c := dst.RGBAAt(x, y)
			if c.R > 220 && c.G > 220 && c.B > 220 {
				whiteCount++
			}
		}
	}

	// If there's a distinct white text block in this area (typical for TRUYENFULL.VN)
	if whiteCount >= 80 && whiteCount <= 1200 {
		inpaintRegion(dst, xStart, yStart, xEnd, yEnd, h)
	}
}

// inpaintRegion fills a rectangular region with smooth vertical & neighbor color interpolation
func inpaintRegion(dst *image.RGBA, x1, y1, x2, y2, totalH int) {
	topY := y1 - 1
	if topY < 0 {
		topY = 0
	}
	botY := y2 + 1
	if botY >= totalH {
		botY = totalH - 1
	}

	for x := x1; x <= x2; x++ {
		topColor := dst.RGBAAt(x, topY)
		botColor := dst.RGBAAt(x, botY)

		for y := y1; y <= y2; y++ {
			alpha := float64(y-y1) / math.Max(1.0, float64(y2-y1))
			r := uint8(float64(topColor.R)*(1.0-alpha) + float64(botColor.R)*alpha)
			g := uint8(float64(topColor.G)*(1.0-alpha) + float64(botColor.G)*alpha)
			b := uint8(float64(topColor.B)*(1.0-alpha) + float64(botColor.B)*alpha)

			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

