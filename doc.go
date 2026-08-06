// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

// Package thumbnail is a pure-Go (CGO_ENABLED=0) implementation of the
// freedesktop.org Thumbnail Managing Standard.
//
// It stores thumbnails under $XDG_CACHE_HOME/thumbnails in the four canonical
// size buckets — normal (128px), large (256px), x-large (512px) and xx-large
// (1024px). Each thumbnail is a PNG whose filename is the MD5 hex digest of the
// canonical file:// URI of the source, and which carries the mandated tEXt
// metadata chunks Thumb::URI and Thumb::MTime (plus optional Thumb::Size,
// Thumb::Mimetype and Software). A cached thumbnail is valid only while its
// recorded Thumb::MTime matches the source file's current modification time;
// otherwise it is regenerated. Sources that cannot be thumbnailed are recorded
// under thumbnails/fail/<appname>/ so they are not retried until they change.
//
// The scaling engine dog-foods the fleet's github.com/go-images/images resizer
// (SIMD bilinear) by default and falls back to golang.org/x/image/draw
// (Catmull-Rom) when a higher-quality filter is requested.
//
// A Provider seam decouples "decode a source into an image" from the caching
// machinery so that non-file sources can be plugged in. Two providers ship: the
// default file-decoding provider, and a live-framebuffer provider that resizes
// an in-memory image.Image directly, bypassing the on-disk cache — intended for
// a compositor's live window / exposé thumbnails.
package thumbnail
