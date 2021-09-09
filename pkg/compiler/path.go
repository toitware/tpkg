package compiler

import (
	"path/filepath"
	"runtime"
	"strings"
)

/*
A compiler path is a path that is recognized by the compiler.
Fundamentally it requires:
- absolute paths must start with '/'
- the segment separator must be a '/'.

These functions must be kept in sync with the one from toitlsp.
*/

type Path string

func ToPath(path string) Path {
	return toCompilerPath(path, runtime.GOOS == "windows")
}

func toCompilerPath(path string, windows bool) Path {
	if !windows {
		return Path(path)
	}
	if filepath.IsAbs(path) {
		path = "/" + path
	}
	return Path(filepath.ToSlash(path))
}

func (path Path) FilePath() string {
	return fromCompilerPath(path, runtime.GOOS == "windows")
}

func fromCompilerPath(path Path, onWindows bool) string {
	p := string(path)
	if !onWindows {
		return p
	}

	p = strings.TrimPrefix(p, "/")
	return filepath.FromSlash(p)
}

// URIPath is a url suitable as a '/' separated path.
// That is, the URL can be used as a path once the '/'s are translated to OS specific
// path-segment separators. Most importantly, such a URL does not contain any `:`.
// If the the escaped URL does not have any scheme (like "https://"), then the
// `string(escapedURLPath)` is a valid URL.
// For example:
// the url 'host.com/c:/foo/bar' is legal, but we wouldn't be able to create
// a folder '.packages/host.com/c:/foo/bar' on Windows, as ':' in paths are not
// allowed there.
// The URIPath fixes this by escaping the ':'.
type URIPath string

// ToURIPath takes a URL and converts it to an URIPath.
// If the given url does not have any scheme (with a ':'), then the returned
// value is a valid URL.
func ToURIPath(url string) URIPath {
	return URIPath(strings.ReplaceAll(url, ":", "%3A"))
}

// URL undoes the escaping done in ToEscapedURLPath.
// If the URL contained other escaped ':', then those are unescaped as well.
func (up URIPath) URL() string {
	return strings.ReplaceAll(string(up), "%3A", ":")
}

func (up URIPath) FilePath() string {
	return filepath.FromSlash(string(up))
}
