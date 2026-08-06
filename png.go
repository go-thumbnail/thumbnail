// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
)

// Standard tEXt keywords defined (or recommended) by the Thumbnail Managing
// Standard.
const (
	KeyURI      = "Thumb::URI"      // canonical source URI (mandatory)
	KeyMTime    = "Thumb::MTime"    // source mtime, decimal seconds (mandatory)
	KeySize     = "Thumb::Size"     // source size in bytes (optional)
	KeyMimetype = "Thumb::Mimetype" // source MIME type (optional)
	KeySoftware = "Software"        // generating software (optional)
)

var pngSignature = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// errBadPNG is returned when a byte stream is not a well-formed PNG.
var errBadPNG = errors.New("thumbnail: not a valid PNG stream")

// pngEncode is a seam over the standard PNG encoder, overridable in tests to
// exercise the encode-error path.
var pngEncode = func(buf *bytes.Buffer, img image.Image) error {
	return (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(buf, img)
}

// encodePNG encodes img to PNG and injects the given tEXt chunks (in ascending
// key order for determinism) immediately before the trailing IEND chunk. text
// maps tEXt keyword to its Latin-1 value.
func encodePNG(img image.Image, text map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	if err := pngEncode(&buf, img); err != nil {
		return nil, err
	}
	raw := buf.Bytes()
	// Locate the IEND chunk (12 bytes: len(0)+"IEND"+crc) at the very end. A
	// successful encode always yields the 8-byte signature plus at least IHDR,
	// IDAT and IEND, so raw is comfortably longer than the 12-byte trailer.
	iend := raw[len(raw)-12:]
	body := raw[:len(raw)-12]

	out := make([]byte, 0, len(raw)+64*len(text))
	out = append(out, body...)
	for _, k := range sortedKeys(text) {
		out = appendTextChunk(out, k, text[k])
	}
	out = append(out, iend...)
	return out, nil
}

// appendTextChunk appends a tEXt chunk (keyword\0value) to dst.
func appendTextChunk(dst []byte, keyword, value string) []byte {
	data := make([]byte, 0, len(keyword)+1+len(value))
	data = append(data, keyword...)
	data = append(data, 0)
	data = append(data, value...)

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(data)))
	dst = append(dst, length[:]...)

	typed := append([]byte("tEXt"), data...)
	dst = append(dst, typed...)

	var crc [4]byte
	binary.BigEndian.PutUint32(crc[:], crc32.ChecksumIEEE(typed))
	dst = append(dst, crc[:]...)
	return dst
}

// readText parses a PNG byte stream and returns all of its tEXt chunks keyed by
// keyword. It returns errBadPNG if the stream is not a valid PNG.
func readText(raw []byte) (map[string]string, error) {
	if !bytes.HasPrefix(raw, pngSignature) {
		return nil, errBadPNG
	}
	out := make(map[string]string)
	p := len(pngSignature)
	for p+8 <= len(raw) {
		length := binary.BigEndian.Uint32(raw[p : p+4])
		ctype := string(raw[p+4 : p+8])
		start := p + 8
		end := start + int(length)
		if end+4 > len(raw) {
			return nil, errBadPNG
		}
		if ctype == "tEXt" {
			data := raw[start:end]
			if i := bytes.IndexByte(data, 0); i >= 0 {
				out[string(data[:i])] = string(data[i+1:])
			}
		}
		p = end + 4 // skip chunk data + CRC
		if ctype == "IEND" {
			break
		}
	}
	return out, nil
}

// sortedKeys returns the keys of m in ascending order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// insertion sort keeps the dependency footprint minimal and is ample for
	// the handful of metadata keys involved.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
