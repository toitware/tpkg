// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"net/url"
	"os"
	"path/filepath"

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

func urlToRelPath(str string) string {
	u, err := url.Parse(str)
	if err != nil {
		// Assume that the urlString is just a normal path.
		return filepath.FromSlash(str)
	}
	return filepath.Join(u.Host, filepath.FromSlash(u.Path))
}