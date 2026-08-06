// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

// Size selects one of the four canonical thumbnail size buckets defined by the
// Thumbnail Managing Standard. The zero value is Normal.
type Size int

const (
	// Normal thumbnails fit within a 128x128 pixel box.
	Normal Size = iota
	// Large thumbnails fit within a 256x256 pixel box.
	Large
	// XLarge thumbnails fit within a 512x512 pixel box.
	XLarge
	// XXLarge thumbnails fit within a 1024x1024 pixel box.
	XXLarge
)

// Pixels returns the maximum edge length, in pixels, of the size bucket.
func (s Size) Pixels() int {
	switch s {
	case Normal:
		return 128
	case Large:
		return 256
	case XLarge:
		return 512
	case XXLarge:
		return 1024
	default:
		return 0
	}
}

// Dir returns the on-disk directory name of the size bucket, as mandated by the
// standard (normal, large, x-large, xx-large).
func (s Size) Dir() string {
	switch s {
	case Normal:
		return "normal"
	case Large:
		return "large"
	case XLarge:
		return "x-large"
	case XXLarge:
		return "xx-large"
	default:
		return ""
	}
}

// Valid reports whether s names one of the four canonical buckets.
func (s Size) Valid() bool {
	return s.Dir() != ""
}
