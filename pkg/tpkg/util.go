// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/toitware/tpkg/pkg/compiler"
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
	return compiler.ToURIPath(url).FilePath()
}

func sdkConstraintToMinSDK(sdk string) (*version.Version, error) {
	if sdk == "" {
		return nil, nil
	}
	if !strings.HasPrefix(sdk, "^") {
		return nil, fmt.Errorf("unexpected sdk constraint: '%s'", sdk)
	}
	minSDK, err := version.NewVersion(sdk[1:])
	if err != nil {
		return nil, err
	}
	return minSDK, nil
}
