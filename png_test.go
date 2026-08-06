// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"testing"
)

func TestEncodeReadTextRoundTrip(t *testing.T) {
	img := solid(3, 2, color.RGBA{10, 20, 30, 255})
	in := map[string]string{
		KeyURI:      "file:///x.png",
		KeyMTime:    "1700000000",
		KeySize:     "42",
		KeySoftware: "go-thumbnail/thumbnail",
	}
	data, err := encodePNG(img, in)
	if err != nil {
		t.Fatal(err)
	}
	// Must still decode as a valid PNG of the right size.
	m, err := decodePNGBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Bounds().Dx() != 3 || m.Bounds().Dy() != 2 {
		t.Fatalf("decoded size = %v", m.Bounds())
	}
	out, err := readText(data)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range in {
		if out[k] != v {
			t.Errorf("text[%q] = %q, want %q", k, out[k], v)
		}
	}
}

func TestEncodePNGError(t *testing.T) {
	orig := pngEncode
	pngEncode = func(*bytes.Buffer, image.Image) error { return errors.New("boom") }
	defer func() { pngEncode = orig }()
	if _, err := encodePNG(solid(1, 1, color.Black), nil); err == nil {
		t.Fatal("expected encode error")
	}
}

func TestReadTextErrors(t *testing.T) {
	// Not a PNG.
	if _, err := readText([]byte("nope")); !errors.Is(err, errBadPNG) {
		t.Errorf("bad prefix: got %v", err)
	}
	// Valid signature but a chunk claiming a length that overruns the buffer.
	buf := append([]byte{}, pngSignature...)
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], 9999)
	buf = append(buf, l[:]...)
	buf = append(buf, []byte("tEXt")...)
	buf = append(buf, 1, 2, 3) // far short of 9999 + CRC
	if _, err := readText(buf); !errors.Is(err, errBadPNG) {
		t.Errorf("truncated: got %v", err)
	}
}

func TestReadTextTEXtWithoutNull(t *testing.T) {
	// A well-formed tEXt chunk whose data contains no NUL separator is skipped,
	// then an IEND terminates the scan.
	buf := append([]byte{}, pngSignature...)
	buf = writeChunk(buf, "tEXt", []byte("nonulseparator"))
	buf = writeChunk(buf, "IEND", nil)
	// Trailing bytes after IEND must be ignored.
	buf = append(buf, 0xDE, 0xAD)
	out, err := readText(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no keys, got %v", out)
	}
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]string{"c": "", "a": "", "b": ""})
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedKeys = %v, want %v", got, want)
		}
	}
}

// decodePNGBytes decodes a PNG from bytes for test assertions.
func decodePNGBytes(data []byte) (image.Image, error) {
	m, _, err := image.Decode(bytes.NewReader(data))
	return m, err
}

// writeChunk appends a full PNG chunk (length, type, data, CRC) to dst.
func writeChunk(dst []byte, ctype string, data []byte) []byte {
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(data)))
	dst = append(dst, l[:]...)
	typed := append([]byte(ctype), data...)
	dst = append(dst, typed...)
	var crc [4]byte
	binary.BigEndian.PutUint32(crc[:], crc32.ChecksumIEEE(typed))
	return append(dst, crc[:]...)
}
