// Copyright (c) the go-thumbnail/thumbnail authors
// SPDX-License-Identifier: BSD-3-Clause

package thumbnail

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
	"path/filepath"
	"strings"
)

// absPath is a seam over filepath.Abs, overridable in tests to exercise the
// (otherwise unreachable) working-directory-resolution failure path.
var absPath = filepath.Abs

// FileURI returns the canonical file:// URI for a local filesystem path, as
// required by the Thumbnail Managing Standard for hash computation. The path is
// made absolute and each path segment is percent-encoded. A path that already
// carries a URI scheme (e.g. "file://…" or "http://…") is returned unchanged.
func FileURI(path string) string {
	if hasScheme(path) {
		return path
	}
	abs, err := absPath(path)
	if err != nil {
		// filepath.Abs only fails when the working directory cannot be
		// determined; fall back to the original path so a URI is still produced.
		abs = path
	}
	abs = filepath.ToSlash(abs)
	segs := strings.Split(abs, "/")
	for i, s := range segs {
		segs[i] = (&url.URL{Path: s}).EscapedPath()
	}
	joined := strings.Join(segs, "/")
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return "file://" + joined
}

// hasScheme reports whether s begins with a URI scheme followed by "://".
func hasScheme(s string) bool {
	i := strings.Index(s, "://")
	if i <= 0 {
		return false
	}
	for _, r := range s[:i] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// Hash returns the lowercase MD5 hex digest of a URI: the basename (without the
// ".png" extension) of the thumbnail file for that URI.
func Hash(uri string) string {
	sum := md5.Sum([]byte(uri))
	return hex.EncodeToString(sum[:])
}

// filenameFor returns the "<md5>.png" thumbnail filename for a URI.
func filenameFor(uri string) string {
	return Hash(uri) + ".png"
}

// localPath returns the local filesystem path for a file:// URI, and reports
// whether uri is in fact a local file URI. Non-file URIs yield ("", false).
func localPath(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return "", false
	}
	return filepath.FromSlash(u.Path), true
}
