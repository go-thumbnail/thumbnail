// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

// srcFile writes a landscape source PNG into a fresh temp dir and returns its
// path plus a Cache rooted at another temp dir.
func srcFile(t *testing.T, size Size, opts ...Option) (string, *Cache) {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.png")
	writePNG(t, src, solid(200, 100, color.RGBA{20, 40, 60, 255}))
	root := t.TempDir()
	all := append([]Option{WithRoot(root)}, opts...)
	return src, New(size, all...)
}

func TestNewDefaults(t *testing.T) {
	xdg.CacheHome = filepath.Join(t.TempDir(), "cache")
	c := New(Normal)
	if c.Root != filepath.Join(xdg.CacheHome, "thumbnails") {
		t.Errorf("default root = %q", c.Root)
	}
	if c.AppName != "go-thumbnail" || c.Software != "go-thumbnail/thumbnail" {
		t.Errorf("defaults appname/software = %q/%q", c.AppName, c.Software)
	}
	if _, ok := c.Provider.(FileProvider); !ok {
		t.Errorf("default provider = %T", c.Provider)
	}
	if c.MaxBytes != defaultMaxBytes {
		t.Errorf("default maxbytes = %d", c.MaxBytes)
	}
}

func TestOptions(t *testing.T) {
	fp := FramebufferProvider{Img: solid(1, 1, color.Black)}
	c := New(Large,
		WithRoot("/root"),
		WithFilter(CatmullRom),
		WithAppName("myapp"),
		WithProvider(fp),
		WithSoftware("sw"),
		WithMaxBytes(123),
	)
	if c.Root != "/root" || c.Filter != CatmullRom || c.AppName != "myapp" ||
		c.Software != "sw" || c.MaxBytes != 123 {
		t.Fatalf("options not applied: %+v", c)
	}
	if _, ok := c.Provider.(FramebufferProvider); !ok {
		t.Fatalf("provider option not applied: %T", c.Provider)
	}
}

func TestPath(t *testing.T) {
	c := New(Normal, WithRoot("/base"))
	uri := FileURI("/foo/x.png")
	want := filepath.Join("/base", "normal", Hash(uri)+".png")
	if got := c.Path("/foo/x.png"); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestGetGeneratesAndCaches(t *testing.T) {
	src, c := srcFile(t, Normal)

	path, err := c.Get(src)
	if err != nil {
		t.Fatal(err)
	}
	// Lives under normal/ with the MD5.png name.
	if filepath.Dir(path) != filepath.Join(c.Root, "normal") {
		t.Errorf("thumb dir = %q", filepath.Dir(path))
	}
	if filepath.Base(path) != Hash(FileURI(src))+".png" {
		t.Errorf("thumb name = %q", filepath.Base(path))
	}

	// The generated thumbnail carries the mandated metadata and correct size.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	txt, err := readText(data)
	if err != nil {
		t.Fatal(err)
	}
	if txt[KeyURI] != FileURI(src) {
		t.Errorf("Thumb::URI = %q", txt[KeyURI])
	}
	if txt[KeyMTime] != mtimeOf(t, src) {
		t.Errorf("Thumb::MTime = %q, want %q", txt[KeyMTime], mtimeOf(t, src))
	}
	if txt[KeySize] == "" || txt[KeySoftware] != "go-thumbnail/thumbnail" {
		t.Errorf("Size/Software = %q/%q", txt[KeySize], txt[KeySoftware])
	}
	if txt[KeyMimetype] != "image/png" {
		t.Errorf("Mimetype = %q, want image/png", txt[KeyMimetype])
	}
	m, err := decodePNGBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Bounds().Dx() != 128 || m.Bounds().Dy() != 64 {
		t.Errorf("thumb dims = %v, want 128x64", m.Bounds())
	}

	// Second call: cache hit (same path, returns the cached decode).
	path2, err := c.Get(src)
	if err != nil || path2 != path {
		t.Fatalf("cache hit path = %q err=%v", path2, err)
	}
}

func TestGetImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big.png")
	writePNG(t, src, solid(600, 300, color.RGBA{20, 40, 60, 255}))
	c := New(Large, WithRoot(t.TempDir()))
	img, err := c.GetImage(src)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 128 {
		t.Fatalf("GetImage bounds = %v", img.Bounds())
	}
	// Second call hits the cached-decode branch.
	if _, err := c.GetImage(src); err != nil {
		t.Fatal(err)
	}
}

func TestStaleRegenerates(t *testing.T) {
	src, c := srcFile(t, Normal)
	if _, err := c.Get(src); err != nil {
		t.Fatal(err)
	}
	// Advance the source mtime so the cached Thumb::MTime no longer matches.
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(src, future, future); err != nil {
		t.Fatal(err)
	}
	path, err := c.Get(src)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	txt, _ := readText(data)
	if txt[KeyMTime] != mtimeOf(t, src) {
		t.Fatalf("stale thumb not regenerated: mtime %q want %q", txt[KeyMTime], mtimeOf(t, src))
	}
}

func TestCorruptCacheRegenerates(t *testing.T) {
	src, c := srcFile(t, Normal)
	// Pre-seed the thumbnail path with garbage; Get must overwrite it.
	tp := c.Path(src)
	if err := os.MkdirAll(filepath.Dir(tp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tp, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(src); err != nil {
		t.Fatal(err)
	}
	if _, err := decodePNGBytes(mustRead(t, tp)); err != nil {
		t.Fatalf("cache not regenerated to valid PNG: %v", err)
	}
}

func TestGetNonFileURI(t *testing.T) {
	c := New(Normal, WithRoot(t.TempDir()))
	if _, err := c.Get("https://example.com/x.png"); err == nil {
		t.Fatal("expected error for non-file URI")
	}
}

func TestGetMissingSource(t *testing.T) {
	c := New(Normal, WithRoot(t.TempDir()))
	if _, err := c.Get(filepath.Join(t.TempDir(), "nope.png")); err == nil {
		t.Fatal("expected stat error for missing source")
	}
}

func TestFailRecordHonored(t *testing.T) {
	// A provider that always fails triggers a fail record; the second call is
	// short-circuited with ErrFailed.
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.png")
	writePNG(t, src, solid(10, 10, color.White))
	c := New(Normal, WithRoot(t.TempDir()), WithProvider(errProvider{}))

	_, err := c.Get(src)
	if err == nil {
		t.Fatal("expected first-attempt error")
	}
	// A fail record must now exist.
	failPath := filepath.Join(c.Root, "fail", c.AppName, Hash(FileURI(src))+".png")
	if _, statErr := os.Stat(failPath); statErr != nil {
		t.Fatalf("fail record not written: %v", statErr)
	}
	// Second attempt is refused with ErrFailed (not retried).
	_, err = c.Get(src)
	if !errors.Is(err, ErrFailed) {
		t.Fatalf("second attempt err = %v, want ErrFailed", err)
	}

	// Touch the source: the fail record is now stale and generation is retried
	// (and fails again the same way, but importantly NOT via ErrFailed short
	// circuit — the provider is invoked).
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(src, future, future); err != nil {
		t.Fatal(err)
	}
	_, err = c.Get(src)
	if err == nil || errors.Is(err, ErrFailed) {
		t.Fatalf("stale fail record should re-invoke provider, got %v", err)
	}
}

func TestScaleFailureRecordsFail(t *testing.T) {
	src, c := srcFile(t, Normal, WithFilter(Filter(99)))
	if _, err := c.Get(src); err == nil {
		t.Fatal("expected scale error")
	}
	failPath := filepath.Join(c.Root, "fail", c.AppName, Hash(FileURI(src))+".png")
	if _, err := os.Stat(failPath); err != nil {
		t.Fatalf("fail record not written on scale failure: %v", err)
	}
}

func TestProviderPanicRecovered(t *testing.T) {
	src, c := srcFile(t, Normal, WithProvider(panicProvider{}))
	_, err := c.Get(src)
	if err == nil || !strings.Contains(err.Error(), "panicked") {
		t.Fatalf("expected recovered panic error, got %v", err)
	}
}

func TestEncodeFailurePropagates(t *testing.T) {
	orig := pngEncode
	pngEncode = func(*bytes.Buffer, image.Image) error { return errors.New("enc") }
	defer func() { pngEncode = orig }()
	src, c := srcFile(t, Normal)
	if _, err := c.Get(src); err == nil {
		t.Fatal("expected encode error to propagate")
	}
}

func TestWriteFailurePropagates(t *testing.T) {
	origMk := mkdirAll
	mkdirAll = func(string, os.FileMode) error { return errors.New("mkdir") }
	defer func() { mkdirAll = origMk }()
	src, c := srcFile(t, Normal)
	if _, err := c.Get(src); err == nil {
		t.Fatal("expected mkdir error to propagate")
	}

	// writeFile error path.
	origWr := writeFile
	writeFile = func(string, []byte, os.FileMode) error { return errors.New("write") }
	defer func() { writeFile = origWr }()
	if _, err := c.Get(src); err == nil {
		t.Fatal("expected write error to propagate")
	}
}

func TestLiveBypassesDisk(t *testing.T) {
	root := t.TempDir()
	c := New(Normal, WithRoot(root))
	fb := FramebufferProvider{Img: solid(200, 100, color.RGBA{1, 2, 3, 255})}
	src, _ := fb.Thumbnail("")
	out, err := c.Live(src)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 128 || out.Bounds().Dy() != 64 {
		t.Fatalf("Live bounds = %v", out.Bounds())
	}
	// Nothing must have been written to the cache root.
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatalf("Live wrote to disk: %v", entries)
	}
}

func TestRecordFailErrors(t *testing.T) {
	c := New(Normal, WithRoot(t.TempDir()))

	// encode error.
	origEnc := pngEncode
	pngEncode = func(*bytes.Buffer, image.Image) error { return errors.New("enc") }
	if err := c.recordFail("/x/y.png", "file:///x", "1"); err == nil {
		t.Error("expected recordFail encode error")
	}
	pngEncode = origEnc

	// write (mkdir) error.
	origMk := mkdirAll
	mkdirAll = func(string, os.FileMode) error { return errors.New("mkdir") }
	if err := c.recordFail("/x/y.png", "file:///x", "1"); err == nil {
		t.Error("expected recordFail write error")
	}
	mkdirAll = origMk
}

func TestMetadataOptional(t *testing.T) {
	// No Software, and an extension with no known MIME type -> both chunks omit.
	c := New(Normal, WithSoftware(""))
	info, _ := os.Stat(filepath.Join("testdata", "landscape.png"))
	m := c.metadata("file:///x.unknownext", "1", info)
	if _, ok := m[KeySoftware]; ok {
		t.Error("Software should be omitted when empty")
	}
	if _, ok := m[KeyMimetype]; ok {
		t.Error("Mimetype should be omitted for unknown extension")
	}
	if m[KeyURI] != "file:///x.unknownext" || m[KeyMTime] != "1" || m[KeySize] == "" {
		t.Errorf("mandatory metadata missing: %v", m)
	}
}

func TestPackageLevelGet(t *testing.T) {
	xdg.CacheHome = t.TempDir()
	dir := t.TempDir()
	src := filepath.Join(dir, "p.png")
	writePNG(t, src, solid(200, 100, color.RGBA{5, 5, 5, 255}))

	path, err := Get(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, xdg.CacheHome) {
		t.Errorf("package Get path %q not under cache home %q", path, xdg.CacheHome)
	}
	img, err := GetImage(src)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 128 {
		t.Errorf("package GetImage bounds = %v", img.Bounds())
	}
}

// --- test doubles ----------------------------------------------------------

type errProvider struct{}

func (errProvider) Thumbnail(string) (image.Image, error) {
	return nil, errors.New("cannot thumbnail")
}

type panicProvider struct{}

func (panicProvider) Thumbnail(string) (image.Image, error) { panic("kaboom") }

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
