// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"os"
	"runtime"
	"testing"

	"github.com/go-gfx/gfx/codec"
)

// claimSize rewrites the IHDR of a PNG to claim w x h, recomputing the IHDR CRC
// (crc32 over the chunk type and data, bytes 12..29, stored at 29).
func claimSize(b []byte, w, h uint32) {
	binary.BigEndian.PutUint32(b[16:], w)
	binary.BigEndian.PutUint32(b[20:], h)
	binary.BigEndian.PutUint32(b[29:], crc32.ChecksumIEEE(b[12:29]))
}

func allocated(f func()) uint64 {
	var a, b runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&a)
	f()
	runtime.ReadMemStats(&b)
	return b.TotalAlloc - a.TotalAlloc
}

// A real PNG whose header claims 16000x16000 made image.Decode allocate about a
// gigabyte. The header check refuses it before the decoder runs.
func TestSourceHeaderBombRefused(t *testing.T) {
	src, c := srcFile(t, Normal)
	b := mustRead(t, src)
	claimSize(b, 16000, 16000)
	if err := os.WriteFile(src, b, 0o644); err != nil {
		t.Fatal(err)
	}
	var err error
	n := allocated(func() { _, err = (FileProvider{}).Thumbnail(src) })
	if !errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	if n >= 1<<20 {
		t.Errorf("provider allocated %d bytes, want < 1 MiB", n)
	}
	t.Logf("provider on the bomb: allocated %d bytes", n)
	// Through the whole Cache (fail record, directories) it is still nothing
	// next to the gigabyte.
	n = allocated(func() { _, err = c.Get(src) })
	if !errors.Is(err, codec.ErrTooLarge) || n >= 4<<20 {
		t.Errorf("Get: err = %v, allocated %d bytes, want < 4 MiB", err, n)
	}
	t.Logf("Cache.Get on the bomb: allocated %d bytes", n)
}

func TestMaxPixelsCeiling(t *testing.T) {
	src, _ := srcFile(t, Normal) // 200x100 = 20000 pixels
	if _, err := (FileProvider{MaxPixels: 20000}).Thumbnail(src); err != nil {
		t.Fatalf("at the ceiling: %v", err)
	}
	if _, err := (FileProvider{MaxPixels: 19999}).Thumbnail(src); !errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("one over: err = %v", err)
	}
	// The knob reaches the default provider through the Cache.
	c := New(Normal, WithRoot(t.TempDir()), WithMaxPixels(19999))
	if _, err := c.Get(src); !errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("WithMaxPixels: err = %v", err)
	}
	c = New(Normal, WithRoot(t.TempDir()), WithMaxPixels(20000))
	if _, err := c.Get(src); err != nil {
		t.Fatalf("WithMaxPixels at ceiling: %v", err)
	}
	if New(Normal).MaxPixels != codec.DefaultMaxPixels {
		t.Errorf("default MaxPixels = %d", New(Normal).MaxPixels)
	}
}

// A whole header and no pixels passes the header check and fails the decode.
func TestTruncatedBodyFailsDecode(t *testing.T) {
	src, _ := srcFile(t, Normal)
	if err := os.WriteFile(src, mustRead(t, src)[:41], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (FileProvider{}).Thumbnail(src); err == nil || errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

// A 42-megapixel source is above the default and refused; raising the ceiling
// is the documented answer.
func TestDefaultCeilingIs40Mpx(t *testing.T) {
	src, _ := srcFile(t, Normal)
	b := mustRead(t, src)
	claimSize(b, 7000, 6000)
	if err := os.WriteFile(src, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (FileProvider{}).Thumbnail(src); !errors.Is(err, codec.ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

// A cached thumbnail whose header claims more than its size bucket is a miss: it
// is regenerated, never decoded.
func TestOversizedCacheFileIsAMiss(t *testing.T) {
	src, c := srcFile(t, Normal)
	tp, err := c.Get(src)
	if err != nil {
		t.Fatal(err)
	}
	b := mustRead(t, tp)
	claimSize(b, 16000, 16000)
	if err := os.WriteFile(tp, b, 0o644); err != nil {
		t.Fatal(err)
	}
	n := allocated(func() { _, err = c.Get(src) })
	if err != nil {
		t.Fatal(err)
	}
	if n >= 16<<20 {
		t.Errorf("allocated %d bytes regenerating, want < 16 MiB", n)
	}
	t.Logf("planted cache bomb: allocated %d bytes", n)
	m, err := decodePNGBytes(mustRead(t, tp))
	if err != nil || m.Bounds().Dx() != 128 {
		t.Fatalf("cache not regenerated: %v %v", m, err)
	}
}
