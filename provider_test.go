// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"
)

func TestFileProvider(t *testing.T) {
	p := FileProvider{}
	img, err := p.Thumbnail(filepath.Join("testdata", "landscape.png"))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 200 || img.Bounds().Dy() != 100 {
		t.Fatalf("decoded bounds = %v", img.Bounds())
	}
}

func TestFileProviderMissing(t *testing.T) {
	if _, err := (FileProvider{}).Thumbnail("testdata/does-not-exist.png"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFileProviderTooBig(t *testing.T) {
	p := FileProvider{MaxBytes: 8}
	if _, err := p.Thumbnail(filepath.Join("testdata", "landscape.png")); err == nil {
		t.Fatal("expected size-limit error")
	}
}

func TestFileProviderBadImage(t *testing.T) {
	if _, err := (FileProvider{}).Thumbnail(filepath.Join("testdata", "broken.png")); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestFramebufferProvider(t *testing.T) {
	img := solid(4, 4, color.RGBA{9, 9, 9, 255})
	p := FramebufferProvider{Img: img}
	got, err := p.Thumbnail("ignored")
	if err != nil {
		t.Fatal(err)
	}
	if got != image.Image(img) {
		t.Fatal("framebuffer provider should return its stored image")
	}
	if _, err := (FramebufferProvider{}).Thumbnail("x"); err == nil {
		t.Fatal("expected error for nil framebuffer image")
	}
}
