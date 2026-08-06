# go-thumbnail

Pure-Go (`CGO_ENABLED=0`) tooling for the freedesktop.org
[Thumbnail Managing Standard](https://specifications.freedesktop.org/thumbnail/latest-single/).

## Repositories

- [**thumbnail**](https://github.com/go-thumbnail/thumbnail) — the standard's
  cache: canonical `$XDG_CACHE_HOME/thumbnails` layout, MD5-of-`file://`-URI
  naming, mandated `Thumb::URI` / `Thumb::MTime` PNG metadata, mtime-based
  validation, `fail/` records, and a pluggable `Provider` seam (default file
  decoder + a live-framebuffer provider for compositor window/exposé
  thumbnails). Resizing dog-foods
  [go-images](https://github.com/go-images/images).

All repositories are BSD-3-Clause licensed, build on `go 1.26.4`, and are
verified in CI at 100% coverage across six 64-bit architectures.

> Org landing page and aggregated docs are a follow-up.
