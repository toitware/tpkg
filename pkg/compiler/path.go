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
