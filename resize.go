// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"fmt"
	"image"

	goimages "github.com/go-images/images"
	xdraw "golang.org/x/image/draw"
)

// Filter selects the scaling kernel used to fit a source image into a bucket.
type Filter int

const (
	// Bilinear scales with the fleet's github.com/go-images/images SIMD bilinear
	// resizer. It is the default.
	Bilinear Filter = iota
	// CatmullRom scales with golang.org/x/image/draw's Catmull-Rom kernel, a
	// higher-quality filter used as the fallback when go-images lacks it.
	CatmullRom
)

// fitBox returns the destination dimensions for scaling a srcW x srcH image so
// it fits within a box x box square while preserving aspect ratio. The source
// is never enlarged: if it already fits, its own dimensions are returned. Both
// returned values are at least 1.
func fitBox(srcW, srcH, box int) (int, int) {
	if srcW <= 0 || srcH <= 0 {
		return 1, 1
	}
	if srcW <= box && srcH <= box {
		return srcW, srcH
	}
	// Scale down by the tighter of the two ratios.
	var w, h int
	if srcW >= srcH {
		w = box
		h = (srcH*box + srcW/2) / srcW
	} else {
		h = box
		w = (srcW*box + srcH/2) / srcH
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// scale fits img into the size bucket using the given filter and returns the
// scaled RGBA image.
func scale(img image.Image, size Size, filter Filter) (*image.RGBA, error) {
	if img == nil {
		return nil, fmt.Errorf("thumbnail: scale: nil image")
	}
	b := img.Bounds()
	w, h := fitBox(b.Dx(), b.Dy(), size.Pixels())

	switch filter {
	case Bilinear:
		// Dog-food the fleet resizer.
		return goimages.Resize(img, w, h, goimages.Bilinear)
	case CatmullRom:
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
		return dst, nil
	default:
		return nil, fmt.Errorf("thumbnail: scale: unknown filter %d", filter)
	}
}
