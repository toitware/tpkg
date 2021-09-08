package path

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

type CompilerPath string

func ToCompilerPath(path string) CompilerPath {
	return toCompilerPath(path, runtime.GOOS == "windows")
}

func toCompilerPath(path string, windows bool) CompilerPath {
	if !windows {
		return CompilerPath(path)
	}
	if filepath.IsAbs(path) {
		path = "/" + path
	}
	return CompilerPath(filepath.ToSlash(path))
}

func (path CompilerPath) ToLocal() string {
	return FromCompilerPath(path)
}

func FromCompilerPath(path CompilerPath) string {
	return fromCompilerPath(path, runtime.GOOS == "windows")
}

func fromCompilerPath(path CompilerPath, onWindows bool) string {
	p := string(path)
	if !onWindows {
		return p
	}

	p = strings.TrimPrefix(p, "/")
	return filepath.FromSlash(p)
}

// EscapedURLPath is a url suitable as a '/' separated path.
// That is, the URL can be used as a path once the '/'s are translated to OS specific
// path-segment separators. Most importantly, such a URL does not contain any `:`.
// If the the escaped URL does not have any scheme (like "https://"), then the
// `string(escapedURLPath)` is a valid URL.
// For example:
// the url 'host.com/c:/foo/bar' is legal, but we wouldn't be able to create
// a folder '.packages/host.com/c:/foo/bar' on Windows, as ':' in paths are not
// allowed there.
// The EscapedURLPath fixes this by escaping the ':'.
type EscapedURLPath string

// ToEscapedURLPath takes a URL and converts it to an EscapedURLPath.
// If the given url does not have any scheme (with a ':'), then the returned
// value is a valid URL.
func ToEscapedURLPath(url string) EscapedURLPath {
	return EscapedURLPath(strings.ReplaceAll(url, ":", "%3A"))
}

// ToURL undoes the escaping done in ToEscapedURLPath.
// If the URL contained other escaped ':', then those are unescaped as well.
func (eurl EscapedURLPath) ToURL() string {
	return strings.ReplaceAll(string(eurl), "%3A", ":")
}
