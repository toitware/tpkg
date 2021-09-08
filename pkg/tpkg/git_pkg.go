// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/toitware/tpkg/pkg/git"
)

func makeContainedReadOnly(dir string, ui UI) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == dir {
			return nil
		}
		if info.IsDir() {
			// Don't change the permissions of directories.
			return nil
		}
		writeBits := uint32(0222)
		info.Mode()
		err = os.Chmod(path, os.FileMode(uint32(info.Mode()) & ^writeBits))
		if err != nil {
			ui.ReportWarning("Error while setting '%s' to read-only: %v", path, err)
		}
		return nil
	})
}

// decomposePkgURL takes a package URL and splits into repository-URL and path.
// The URL can be used to check out the repository, and the path then points to
// the package in the repository.
// For example `github.com/toitware/test-pkg.git/bar/gee` is decomposed into
// `github.com/toitware/test-pkg` and `bar/gee`
func decomposePkgURL(url string) (string, string) {
	if lastIndex := strings.LastIndex(url, ".git/"); lastIndex >= 0 {
		path := url[lastIndex+len(".git/"):]
		url = url[:lastIndex]
		return url, path
	}
	return url, ""
}

// DownloadGit downloads a package, defined by [url] and [version] into the given
// [dir].
// If the [dir] exists it will first be removed to erase old data.
// This function might create an adjacent directory first. For example, if the target
// is `download/here`, then this function might first create a `download/tmp` directory.
// Returns the checked-out hash.
func DownloadGit(ctx context.Context, dir string, urlStr string, version string, hash string, ui UI) (string, error) {
	_, err := os.Stat(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	} else if err != nil {
		err = os.RemoveAll(dir)
		if err != nil {
			return "", ui.ReportError("Failed to remove old package directory '%s': %v", dir, err)
		}
	}

	cloneURL := urlStr
	path := ""
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	checkoutDir := dir

	if strings.Contains(urlStr, "path.toit.io") {
		for _, c := range urlStr {
			print(string(c))
			print(" ")
		}
		println("")
	}
	// If the url's host is 'path.toit.io', then we know that the URL's path
	// should be used as file path.
	// Otherwise we assume it's a https-URL.
	if strings.HasPrefix(urlStr, "path.toit.io/") {
		println("recognized prefix")
		cloneURL = strings.TrimPrefix(urlStr, "path.toit.io/")
		path = urlStr
	} else {
		cloneURL, path = decomposePkgURL(urlStr)

		if path != "" {
			lastSegment := path[strings.LastIndex(path, "/")+1:] // Note that this also works if there isn't any '/'.
			tag = lastSegment + "-v" + version
			// Download into a directory adjacent to the final target.
			baseDir := filepath.Dir(dir)
			err = os.MkdirAll(baseDir, 0755)
			if err != nil {
				return "", err
			}
			// The checkout directory must be on the same drive as the final target, as we are using a
			// rename-command to move the nested package to its final position.
			checkoutDir, err = ioutil.TempDir(baseDir, "partial-toit-checkout")
			if err != nil {
				return "", ui.ReportError("Failed to create temporary directory to download '%s - %s': %v", urlStr, version, err)
			}
			defer os.RemoveAll(checkoutDir)
		}
	}

	err = os.MkdirAll(checkoutDir, 0755)
	if err != nil {
		return "", ui.ReportError("Failed to create download directory '%s': %v", checkoutDir, err)
	}
	successfullyDownloaded := false
	defer func() {
		if !successfullyDownloaded {
			// Try not to leave partially downloaded packages around.
			os.RemoveAll(checkoutDir)
		}
	}()

	downloadedHash, err := git.Clone(ctx, checkoutDir, &git.CloneOptions{
		URL:          cloneURL,
		SingleBranch: true,
		Depth:        1,
		Tag:          tag,
		Hash:         hash,
	})

	if err != nil {
		return "", ui.ReportError("Error while cloning '%s' with tag '%s': %v", urlStr, tag, err)
	}

	if checkoutDir == dir {
		makeContainedReadOnly(dir, ui)
		successfullyDownloaded = true
		return downloadedHash, nil
	}
	// We still need to move the package into its correct location.

	nestedPath := filepath.Join(checkoutDir, filepath.FromSlash(path))
	stat, err := os.Stat(nestedPath)
	if os.IsNotExist(err) {
		return "", ui.ReportError("Repository '%s' does not have path '%s'", urlStr, path)
	} else if err != nil {
		return "", err
	} else if !stat.IsDir() {
		return "", ui.ReportError("Path '%s' in repository '%s' is not a directory", path, urlStr)
	}

	// Renaming only works when the two locations are on the same drive. This is why we didn't
	// check out into a /tmp directory, but checked out in an adjacent directory instead.
	err = os.Rename(nestedPath, dir)
	if err != nil {
		return "", ui.ReportError("Failed to move nested package '%s' to its location '%s'", nestedPath, dir)
	}

	makeContainedReadOnly(dir, ui)
	successfullyDownloaded = true
	return downloadedHash, nil
}
