# thumbnail

[![CI](https://github.com/go-thumbnail/thumbnail/actions/workflows/ci.yml/badge.svg)](https://github.com/go-thumbnail/thumbnail/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-thumbnail/thumbnail.svg)](https://pkg.go.dev/github.com/go-thumbnail/thumbnail)
[![License: BSD-3-Clause](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)
[![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](#tests)

A pure-Go (`CGO_ENABLED=0`) implementation of the freedesktop.org
[**Thumbnail Managing Standard**](https://specifications.freedesktop.org/thumbnail/latest-single/).

It generates and caches file thumbnails exactly where and how the standard
prescribes, so thumbnails written by this library are visible to other
compliant tools (file managers, image viewers) and vice versa.

## What it does

- **Canonical cache layout** — thumbnails live under
  `$XDG_CACHE_HOME/thumbnails/{normal,large,x-large,xx-large}`, the
  128 / 256 / 512 / 1024-pixel buckets. `$XDG_CACHE_HOME` is resolved with
  [`github.com/adrg/xdg`](https://github.com/adrg/xdg).
- **MD5 naming** — a thumbnail's filename is the MD5 hex digest of the
  canonical `file://` URI of its source, plus `.png`.
- **Mandated PNG metadata** — every thumbnail carries `tEXt` chunks
  `Thumb::URI` and `Thumb::MTime` (and optional `Thumb::Size`,
  `Thumb::Mimetype`, `Software`).
- **Validation & invalidation** — a cached thumbnail is served only while its
  stored `Thumb::MTime` matches the source file's current mtime; otherwise it
  is transparently regenerated.
- **Fail records** — sources that cannot be thumbnailed are recorded under
  `thumbnails/fail/<appname>/` and are not retried until the source changes.
- **Fleet resizer** — scaling dog-foods
  [`github.com/go-images/images`](https://github.com/go-images/images) (SIMD
  bilinear) by default, with `golang.org/x/image/draw` (Catmull-Rom) available
  as a higher-quality fallback filter. Aspect ratio is always preserved and
  sources are never upscaled.
- **Provider seam** — a `Provider` interface decouples "decode a source into an
  image" from the caching machinery, so non-file sources can be plugged in. A
  **live-framebuffer** provider resizes an in-memory `image.Image` directly
  (bypassing the disk cache) for a compositor's live window / exposé thumbnails.

## Install

```sh
go get github.com/go-thumbnail/thumbnail
```

## Quickstart

```go
package main

import (
	"fmt"

	"github.com/go-thumbnail/thumbnail"
)

func main() {
	// Package-level helpers use a default normal-size (128px) cache.
	path, err := thumbnail.Get("/home/jens/photo/me.png")
	if err != nil {
		panic(err)
	}
	fmt.Println("cached thumbnail at", path)

	// A configured cache: 256px bucket, custom fail-dir app name.
	c := thumbnail.New(thumbnail.Large, thumbnail.WithAppName("myapp"))
	img, err := c.GetImage("/home/jens/photo/me.png")
	if err != nil {
		panic(err)
	}
	fmt.Println("thumbnail bounds:", img.Bounds())
}
```

### Live framebuffer (compositor window / exposé thumbnails)

```go
c := thumbnail.New(thumbnail.Normal)

// src is an image.Image you already hold — a window's framebuffer, say.
// Live resizes into the bucket and never touches the on-disk cache.
scaled, err := c.Live(src)
```

## API

| Symbol | Purpose |
| ------ | ------- |
| `Get(uri) (string, error)` | Path to the cached (normal) thumbnail, generating if missing/stale. |
| `GetImage(uri) (image.Image, error)` | The thumbnail image itself. |
| `New(size, ...Option) *Cache` | A cache bound to a size bucket. |
| `Cache.Get / GetImage / Path` | Cached thumbnail path / image / would-be path. |
| `Cache.Live(image.Image)` | Resize an in-memory image into the bucket, no disk I/O. |
| `Provider` | `Thumbnail(src string) (image.Image, error)` — pluggable source decoder. |
| `FileProvider`, `FramebufferProvider` | The default file decoder and the live-framebuffer seam. |
| `Size` | `Normal`, `Large`, `XLarge`, `XXLarge`. |
| `Filter` | `Bilinear` (go-images), `CatmullRom` (x/image/draw). |

Options: `WithRoot`, `WithFilter`, `WithAppName`, `WithProvider`,
`WithSoftware`, `WithMaxBytes`.

## Scope

This library implements the *Thumbnail Managing Standard* — the cache layout,
naming, metadata, validation and fail-record rules, plus a resize pipeline over
the standard-library image decoders (PNG, JPEG, GIF). It does **not** implement
the companion *thumbnailer* D-Bus service or `.thumbnailer` provider discovery;
non-image sources are handled by supplying your own `Provider`.

## Tests

100% statement coverage (including every error branch), verified in CI along
with `go vet` and a nine-target cross-build (linux amd64/arm64/riscv64/ppc64le/
s390x/loong64, darwin amd64/arm64, windows amd64).

```sh
go test -covermode=set -coverprofile=cover.out ./...
go tool cover -func=cover.out
```

## License

BSD-3-Clause — see [LICENSE](LICENSE).
