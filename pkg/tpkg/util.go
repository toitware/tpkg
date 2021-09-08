// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"os"
	"path/filepath"

	"github.com/toitware/tpkg/pkg/path"
)

func isDirectory(p string) (bool, error) {
	stat, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	return stat.IsDir(), nil
}

func isFile(p string) (bool, error) {
	info, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if info.IsDir() {
		return false, nil
	}
	return true, nil
}

func URLVersionToRelPath(url string, version string) string {
	return filepath.Join(urlToRelPath(url), version)
}

func urlToRelPath(url string) string {
	escaped := string(path.ToEscapedURLPath(url))
	return filepath.FromSlash(escaped)
}
