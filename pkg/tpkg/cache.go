// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Cache handles all package-Cache related functionality.
// It keeps track of where the caches are, and how to compute paths for packages.
type Cache struct {
	// The locations where packages can be found.
	// The first path is used to download new packages that don't exist yet.
	pkgCachePaths []string
	// The locations where git registries can be found.
	// The first path is used to install new git registries.
	registryCachePaths []string

	ui UI
}

func NewCache(pkgCachePaths []string, registryCachePaths []string, ui UI) Cache {
	return Cache{
		pkgCachePaths:      pkgCachePaths,
		registryCachePaths: registryCachePaths,
		ui:                 ui,
	}
}

func (c Cache) find(p string, paths []string) (string, error) {
	for _, cachePath := range paths {
		cachePath := filepath.Join(cachePath, p)
		info, err := os.Stat(cachePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", c.ui.ReportError("Path %s exists but is not a directory", p)
		}
		return cachePath, nil
	}
	return "", nil
}

// FindPkg searches for the path of 'url'-'version' in the cache.
// If it's not found returns "".
func (c Cache) FindPkg(rootPath string, url string, version string) (string, error) {
	packageRel := URLVersionToRelPath(url, version)
	fullProjectPackagesPath := filepath.Join(rootPath, ProjectPackagesPath)
	return c.find(packageRel, append([]string{fullProjectPackagesPath}, c.pkgCachePaths...))
}

// FindRegistry searches for the path of the registry with the given url in the cache.
// If it's not found returns "".
func (c Cache) FindRegistry(url string) (string, error) {
	registryRel := urlToRelPath(url)
	return c.find(registryRel, c.registryCachePaths)
}

// Returns the path to the specification of the package url-version.
// If the package is not in the cache returns "".
func (c Cache) SpecPathFor(projectRootPath string, url string, version string) (string, error) {
	pkgPath, err := c.FindPkg(projectRootPath, url, version)
	if err != nil {
		return "", err
	}
	if pkgPath == "" {
		return "", nil
	}
	specPath := filepath.Join(pkgPath, DefaultSpecName)
	ok, err := isFile(specPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("missing spec file for package '%s/%s'", url, version)
	}
	return specPath, nil
}

// PreferredPkgPath returns the preferred path for the package url-version.
func (c Cache) PreferredPkgPath(projectRootPath string, url string, version string) string {
	packageRel := URLVersionToRelPath(url, version)
	return filepath.Join(projectRootPath, ProjectPackagesPath, packageRel)
}

// PreferredRegistryPath returns the preferred path for the given registry url.
func (c Cache) PreferredRegistryPath(url string) string {
	// The first cache path is the preferred location.
	return filepath.Join(c.registryCachePaths[0], url)
}

const readmeContent string = `# Package Cache Directory

This directory contains Toit packages that have been downloaded by
the Toit package management system.

Generally, the package manager is able to download these packages again. It
is thus safe to remove the content of this directory.
`

// CreatePackagesCacheDir creates the package cache dir.
// If the directory doesn't exist yet, creates it, and writes a README
// explaining what the directory is for, and what is allowed to be done.
func (c Cache) CreatePackagesCacheDir(projectRootPath string, ui UI) error {
	packagesCacheDir := filepath.Join(projectRootPath, ProjectPackagesPath)
	stat, err := os.Stat(packagesCacheDir)
	if err == nil && !stat.IsDir() {
		return ui.ReportError("Package cache path already exists but is not a directory: '%s'", packagesCacheDir)
	}
	if !os.IsNotExist(err) {
		return err
	}
	err = os.Mkdir(packagesCacheDir, 0755)
	if err != nil {
		return err
	}
	readmePath := filepath.Join(packagesCacheDir, "README.md")
	err = ioutil.WriteFile(readmePath, []byte(readmeContent), 0755)
	return err
}
