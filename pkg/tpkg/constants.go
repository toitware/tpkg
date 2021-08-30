// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

const (
	// ProjectPackagesPath provides the path, relative to the project's root,
	// into which packages should be downloaded.
	ProjectPackagesPath = ".packages"

	DefaultSpecName     = "package.yaml"
	DefaultLockFileName = "package.lock"

	// The directory inside registries, where descriptions should be stored.
	PackageDescriptionDir = "packages"

	// The default filename for description files.
	DescriptionFileName = "desc.yaml"
)
