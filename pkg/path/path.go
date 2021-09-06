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
