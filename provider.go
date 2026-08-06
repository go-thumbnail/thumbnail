// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	// Register the standard decoders used by the default file provider.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// Provider decodes or synthesizes the full-resolution source image identified
// by src. The default file provider treats src as a filesystem path; other
// providers may interpret it however they wish. A Provider must never panic on
// malformed input — it returns an error instead — but the cache also guards
// against panics defensively.
type Provider interface {
	Thumbnail(src string) (image.Image, error)
}

// FileProvider is the default Provider: it decodes an image file at the given
// path using the standard library's registered decoders (PNG, JPEG, GIF). If
// MaxBytes is positive, sources whose on-disk size exceeds MaxBytes are rejected
// before decoding to bound resource use.
type FileProvider struct {
	MaxBytes int64
}

// Thumbnail reads and decodes the image file at path.
func (p FileProvider) Thumbnail(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if p.MaxBytes > 0 && int64(len(data)) > p.MaxBytes {
		return nil, fmt.Errorf("thumbnail: source %q is %d bytes, exceeds limit %d",
			path, len(data), p.MaxBytes)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// FramebufferProvider is a live-framebuffer Provider seam for callers such as a
// compositor that already hold an image in memory (a window's framebuffer, an
// exposé snapshot). Src is ignored; the stored image is returned directly. Used
// with Cache.Live, it resizes without ever touching the on-disk cache.
type FramebufferProvider struct {
	// Img is the live source image. A nil Img yields an error.
	Img image.Image
}

// Thumbnail returns the stored framebuffer image, ignoring src.
func (p FramebufferProvider) Thumbnail(_ string) (image.Image, error) {
	if p.Img == nil {
		return nil, fmt.Errorf("thumbnail: framebuffer provider has no image")
	}
	return p.Img, nil
}
