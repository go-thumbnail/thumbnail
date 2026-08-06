// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"image"
	"image/color"
	"testing"
)

func TestFitBox(t *testing.T) {
	cases := []struct {
		w, h, box    int
		wantW, wantH int
	}{
		{200, 100, 128, 128, 64},  // landscape scales down, longest side = box
		{100, 200, 128, 64, 128},  // portrait
		{64, 64, 128, 64, 64},     // smaller than box -> unchanged (no upscale)
		{256, 256, 128, 128, 128}, // square scales to box
		{0, 10, 128, 1, 1},        // degenerate -> clamped to 1x1
		{4000, 1, 128, 128, 1},    // extreme landscape -> height clamps to 1
		{1, 4000, 128, 1, 128},    // extreme portrait -> width clamps to 1
	}
	for _, c := range cases {
		w, h := fitBox(c.w, c.h, c.box)
		if w != c.wantW || h != c.wantH {
			t.Errorf("fitBox(%d,%d,%d) = %d,%d want %d,%d",
				c.w, c.h, c.box, w, h, c.wantW, c.wantH)
		}
	}
}

func TestScale(t *testing.T) {
	src := solid(200, 100, color.RGBA{1, 2, 3, 255})

	// Bilinear (go-images) fits into the Normal 128 box preserving aspect.
	got, err := scale(src, Normal, Bilinear)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 128 || got.Bounds().Dy() != 64 {
		t.Fatalf("bilinear size = %v", got.Bounds())
	}

	// CatmullRom (x/image/draw fallback) downscales to the same fitted target.
	got, err = scale(src, Normal, CatmullRom)
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds().Dx() != 128 || got.Bounds().Dy() != 64 {
		t.Fatalf("catmullrom size = %v", got.Bounds())
	}
}

func TestScaleErrors(t *testing.T) {
	if _, err := scale(nil, Normal, Bilinear); err == nil {
		t.Error("expected error for nil image")
	}
	if _, err := scale(image.NewRGBA(image.Rect(0, 0, 4, 4)), Normal, Filter(99)); err == nil {
		t.Error("expected error for unknown filter")
	}
}
