// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// --- helpers ---------------------------------------------------------------

// solid returns a w x h image filled with a single colour.
func solid(w, h int, c color.Color) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.Set(x, y, c)
		}
	}
	return m
}

// writePNG encodes img (with no metadata) to path.
func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// mtimeOf returns the decimal-second mtime string of path.
func mtimeOf(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return strconv.FormatInt(info.ModTime().Unix(), 10)
}

// --- Size ------------------------------------------------------------------

func TestSize(t *testing.T) {
	cases := []struct {
		s     Size
		px    int
		dir   string
		valid bool
	}{
		{Normal, 128, "normal", true},
		{Large, 256, "large", true},
		{XLarge, 512, "x-large", true},
		{XXLarge, 1024, "xx-large", true},
		{Size(99), 0, "", false},
	}
	for _, c := range cases {
		if got := c.s.Pixels(); got != c.px {
			t.Errorf("Size(%d).Pixels() = %d, want %d", c.s, got, c.px)
		}
		if got := c.s.Dir(); got != c.dir {
			t.Errorf("Size(%d).Dir() = %q, want %q", c.s, got, c.dir)
		}
		if got := c.s.Valid(); got != c.valid {
			t.Errorf("Size(%d).Valid() = %v, want %v", c.s, got, c.valid)
		}
	}
}

// --- URI / hashing ---------------------------------------------------------

func TestHashKnownVector(t *testing.T) {
	// Canonical MD5-of-URI vector.
	const uri = "file:///home/jens/photo/me.png"
	const want = "d40775e596682f2a16d1b834c221c0a2"
	if got := Hash(uri); got != want {
		t.Fatalf("Hash(%q) = %q, want %q", uri, got, want)
	}
	if got := filenameFor(uri); got != want+".png" {
		t.Fatalf("filenameFor(%q) = %q, want %q", uri, got, want+".png")
	}
}

func TestFileURI(t *testing.T) {
	// Absolute path with a space -> percent-encoded segment.
	if got := FileURI("/foo/bar baz.png"); got != "file:///foo/bar%20baz.png" {
		t.Errorf("FileURI space = %q", got)
	}
	// Already a URI -> unchanged.
	const u = "file:///already/uri.png"
	if got := FileURI(u); got != u {
		t.Errorf("FileURI passthrough = %q", got)
	}
	if got := FileURI("https://example.com/x.png"); got != "https://example.com/x.png" {
		t.Errorf("FileURI http passthrough = %q", got)
	}
	// Relative path is made absolute (prefixed with cwd, so starts with file:///).
	if got := FileURI("rel.png"); got[:8] != "file:///" {
		t.Errorf("FileURI relative = %q, want file:/// prefix", got)
	}
}

func TestFileURIAbsError(t *testing.T) {
	orig := absPath
	absPath = func(string) (string, error) { return "", errors.New("no cwd") }
	defer func() { absPath = orig }()
	// Falls back to the original (relative) path, still yielding a file URI.
	if got := FileURI("rel.png"); got != "file:///rel.png" {
		t.Errorf("FileURI with abs error = %q, want file:///rel.png", got)
	}
}

func TestHasScheme(t *testing.T) {
	cases := map[string]bool{
		"file:///x":         true,
		"http://x":          true,
		"HTTPS://x":         true,
		"a+b-c.d://x":       true,
		"/plain/path":       false,
		"://noscheme":       false,
		"bad scheme://x":    false, // space is not a valid scheme char
		"C:\\windows\\path": false,
		"rel.png":           false,
	}
	for in, want := range cases {
		if got := hasScheme(in); got != want {
			t.Errorf("hasScheme(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLocalPath(t *testing.T) {
	p, ok := localPath("file:///tmp/x.png")
	if !ok || p != filepath.FromSlash("/tmp/x.png") {
		t.Errorf("localPath file = %q,%v", p, ok)
	}
	if _, ok := localPath("https://example.com/x.png"); ok {
		t.Error("localPath http should be false")
	}
	if _, ok := localPath("://%zz"); ok {
		t.Error("localPath unparseable should be false")
	}
}
