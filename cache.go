// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"mime"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/xdg"
)

// defaultMaxBytes bounds the size of a source file the default provider will
// decode, guarding against pathological inputs. Callers may raise or remove the
// limit with WithMaxBytes.
const defaultMaxBytes = 100 << 20 // 100 MiB

// ErrFailed is wrapped by the error returned from Get/GetImage when the source
// is recorded in the fail directory for the current mtime, meaning a previous
// attempt failed and the source has not changed since.
var ErrFailed = errors.New("thumbnail: source previously failed to thumbnail")

// Seams over the standard library, overridable in tests so that
// otherwise-unreachable I/O error paths can be exercised.
var (
	mkdirAll  = os.MkdirAll
	writeFile = os.WriteFile
	readFile  = os.ReadFile
	statFn    = os.Stat
)

// Cache is a thumbnail cache bound to a single size bucket. Construct it with
// New. The exported fields may be read for introspection; mutate them only
// before first use.
type Cache struct {
	// Root is the base thumbnails directory (default $XDG_CACHE_HOME/thumbnails).
	Root string
	// Size is the size bucket this cache serves.
	Size Size
	// Filter selects the scaling kernel (default Bilinear via go-images).
	Filter Filter
	// AppName names the fail subdirectory (thumbnails/fail/<AppName>).
	AppName string
	// Provider decodes sources into images (default FileProvider).
	Provider Provider
	// Software is written to the Software tEXt chunk; empty omits it.
	Software string
	// MaxBytes bounds the default provider's source file size.
	MaxBytes int64
}

// Option configures a Cache in New.
type Option func(*Cache)

// WithFilter sets the scaling filter.
func WithFilter(f Filter) Option { return func(c *Cache) { c.Filter = f } }

// WithAppName sets the fail-directory application name.
func WithAppName(name string) Option { return func(c *Cache) { c.AppName = name } }

// WithProvider sets a custom source Provider.
func WithProvider(p Provider) Option { return func(c *Cache) { c.Provider = p } }

// WithRoot overrides the base thumbnails directory.
func WithRoot(root string) Option { return func(c *Cache) { c.Root = root } }

// WithSoftware sets the Software tEXt value (empty string omits the chunk).
func WithSoftware(s string) Option { return func(c *Cache) { c.Software = s } }

// WithMaxBytes bounds the default provider's source file size (0 disables it).
func WithMaxBytes(n int64) Option { return func(c *Cache) { c.MaxBytes = n } }

// defaultRoot returns $XDG_CACHE_HOME/thumbnails, read at call time.
func defaultRoot() string { return filepath.Join(xdg.CacheHome, "thumbnails") }

// New returns a Cache for the given size bucket with the supplied options
// applied. Unset options take fleet-standard defaults.
func New(size Size, opts ...Option) *Cache {
	c := &Cache{
		Size:     size,
		AppName:  "go-thumbnail",
		Software: "go-thumbnail/thumbnail",
		MaxBytes: defaultMaxBytes,
	}
	for _, o := range opts {
		o(c)
	}
	if c.Root == "" {
		c.Root = defaultRoot()
	}
	if c.Provider == nil {
		c.Provider = FileProvider{MaxBytes: c.MaxBytes}
	}
	return c
}

// Get returns the path to the cached normal-size thumbnail for uri, using a
// default cache rooted at $XDG_CACHE_HOME/thumbnails. It generates and caches
// the thumbnail if absent or stale.
func Get(uri string) (string, error) { return New(Normal).Get(uri) }

// GetImage returns the normal-size thumbnail image for uri, using a default
// cache rooted at $XDG_CACHE_HOME/thumbnails.
func GetImage(uri string) (image.Image, error) { return New(Normal).GetImage(uri) }

// Path returns the on-disk path where the thumbnail for uri would live, whether
// or not it currently exists. uri may be a filesystem path or a URI.
func (c *Cache) Path(uri string) string {
	return filepath.Join(c.Root, c.Size.Dir(), filenameFor(FileURI(uri)))
}

// Get returns the path to the cached thumbnail for uri, generating and caching
// it if absent or stale. uri may be a filesystem path or a file:// URI.
func (c *Cache) Get(uri string) (string, error) {
	path, _, err := c.get(uri)
	return path, err
}

// GetImage returns the thumbnail image for uri, generating and caching it if
// absent or stale.
func (c *Cache) GetImage(uri string) (image.Image, error) {
	_, img, err := c.get(uri)
	return img, err
}

// Live resizes an in-memory image into this cache's size bucket without
// touching the on-disk cache. It is the live-framebuffer entry point used for a
// compositor's window and exposé thumbnails.
func (c *Cache) Live(img image.Image) (*image.RGBA, error) {
	return scale(img, c.Size, c.Filter)
}

// get implements the full cached-thumbnail lifecycle for uri.
func (c *Cache) get(uri string) (string, image.Image, error) {
	uri = FileURI(uri)
	name := filenameFor(uri)
	thumbPath := filepath.Join(c.Root, c.Size.Dir(), name)

	path, ok := localPath(uri)
	if !ok {
		return "", nil, fmt.Errorf("thumbnail: Get requires a file:// source, got %q", uri)
	}
	info, err := statFn(path)
	if err != nil {
		return "", nil, err
	}
	mtime := strconv.FormatInt(info.ModTime().Unix(), 10)

	// Honor a matching fail record so failures are not retried until the source
	// changes.
	failPath := filepath.Join(c.Root, "fail", c.AppName, name)
	if data, ferr := readFile(failPath); ferr == nil {
		if txt, terr := readText(data); terr == nil && txt[KeyMTime] == mtime {
			return "", nil, fmt.Errorf("%w: %s", ErrFailed, uri)
		}
	}

	// Serve a still-valid cached thumbnail.
	if data, rerr := readFile(thumbPath); rerr == nil {
		if txt, terr := readText(data); terr == nil && txt[KeyMTime] == mtime {
			if img, derr := png.Decode(bytes.NewReader(data)); derr == nil {
				return thumbPath, img, nil
			}
		}
	}

	// Generate.
	src, err := c.produce(path)
	if err != nil {
		_ = c.recordFail(failPath, uri, mtime)
		return "", nil, err
	}
	scaled, err := scale(src, c.Size, c.Filter)
	if err != nil {
		_ = c.recordFail(failPath, uri, mtime)
		return "", nil, err
	}
	data, err := encodePNG(scaled, c.metadata(uri, mtime, info))
	if err != nil {
		return "", nil, err
	}
	if err := c.writeThumb(thumbPath, data); err != nil {
		return "", nil, err
	}
	return thumbPath, scaled, nil
}

// produce invokes the provider, converting any panic into an error so a
// misbehaving provider can never crash the caller.
func (c *Cache) produce(src string) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("thumbnail: provider panicked: %v", r)
		}
	}()
	return c.Provider.Thumbnail(src)
}

// metadata builds the tEXt chunk set for a generated thumbnail.
func (c *Cache) metadata(uri, mtime string, info os.FileInfo) map[string]string {
	m := map[string]string{
		KeyURI:   uri,
		KeyMTime: mtime,
		KeySize:  strconv.FormatInt(info.Size(), 10),
	}
	if mt := mime.TypeByExtension(filepath.Ext(uri)); mt != "" {
		m[KeyMimetype] = mt
	}
	if c.Software != "" {
		m[KeySoftware] = c.Software
	}
	return m
}

// writeThumb writes data to path, creating parent directories (0700) and the
// file (0600) per the standard's privacy requirements.
func (c *Cache) writeThumb(path string, data []byte) error {
	if err := mkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeFile(path, data, 0o600)
}

// recordFail writes a minimal fail marker (a 1x1 PNG carrying the source URI and
// mtime) so the failed source is not retried until it changes.
func (c *Cache) recordFail(failPath, uri, mtime string) error {
	marker := image.NewRGBA(image.Rect(0, 0, 1, 1))
	meta := map[string]string{KeyURI: uri, KeyMTime: mtime}
	if c.Software != "" {
		meta[KeySoftware] = c.Software
	}
	data, err := encodePNG(marker, meta)
	if err != nil {
		return err
	}
	return c.writeThumb(failPath, data)
}
