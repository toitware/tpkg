// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// pkgPathForDir returns the pkg-init file in the given directory.
// The given dir must not be empty.
func pkgPathForDir(dir string) string {
	if dir == "" {
		log.Fatal("Directory must not be empty")
	}
	return filepath.Join(dir, DefaultSpecName)
}

// lockPathForDir returns the lock file in the given directory.
// The given dir must not be empty.
func lockPathForDir(dir string) string {
	if dir == "" {
		log.Fatal("Directory must not be empty")
	}

	return filepath.Join(dir, DefaultLockFileName)
}

// InitDirectory initializes the project root as the root for a package or application.
// If no root is given, initializes the current directory instead.
func InitDirectory(projectRoot string, ui UI) error {
	dir := projectRoot
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = cwd
	}
	pkgPath := pkgPathForDir(dir)
	lockPath := lockPathForDir(dir)

	pkgExists, err := isFile(pkgPath)
	if err != nil {
		return err
	}
	lockExists, err := isFile(lockPath)
	if err != nil {
		return err
	}

	if pkgExists || lockExists {
		ui.ReportInfo("Directory '%s' already initialized", dir)
		return nil
	}
	err = ioutil.WriteFile(pkgPath, []byte("# Toit Package File.\n"), 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(lockPath, []byte("# Toit Lock File.\n"), 0644)
}
