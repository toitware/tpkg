// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tracking"
)

type ProjectPaths struct {
	// Project root.
	ProjectRootPath string

	// The path of the lock file for the current project.
	LockFile string

	// The path of the spec file for the current project.
	SpecFile string
}

// Manager serves as entry point for all package-management related operations.
// Use NewManager to create a new manager.
type Manager struct {
	// The loaded registries.
	registries Registries

	// The package cache.
	cache Cache

	// The UI to communicate with the user.
	ui UI

	track tracking.Track
}

// ProjectPackageManager: a package manager for a specific project.
type ProjectPkgManager struct {
	*Manager

	// The project relevant Paths.
	Paths *ProjectPaths
}

// DescRegistry combines a description with the registry it comes from.
type DescRegistry struct {
	Desc     *Desc
	Registry Registry
}

type DescRegistries []DescRegistry

// NewManager returns a new Manager.
func NewManager(registries Registries, cache Cache, ui UI, track tracking.Track) *Manager {
	return &Manager{
		registries: registries,
		cache:      cache,
		ui:         ui,
		track:      track,
	}
}

func NewProjectPkgManager(manager *Manager, paths *ProjectPaths) *ProjectPkgManager {
	return &ProjectPkgManager{
		Manager: manager,
		Paths:   paths,
	}
}

// prepareInstallLocal prepares the installation of a local package.
// It verifies that the path is valid.
// Returns a suggested prefix.
func (m *Manager) prepareInstallLocal(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if isDir, err := isDirectory(abs); !isDir || err != nil {
		if err == nil {
			return "", m.ui.ReportError("Target '%s' is not a directory", path)
		}
		return "", m.ui.ReportError("Target '%s' is not a directory: %v", path, err)
	}
	prefix := filepath.Base(abs)
	return prefix, nil
}

// download fetches the given url/version, unless it's already in the cache.
func (m *ProjectPkgManager) download(ctx context.Context, url string, version string, hash string) error {
	projectRoot := m.Paths.ProjectRootPath
	packagePath, err := m.cache.FindPkg(projectRoot, url, version)
	if err != nil {
		return err
	}
	if packagePath != "" {
		return nil
	}
	err = m.cache.CreatePackagesCacheDir(projectRoot, m.ui)
	if err != nil {
		return err
	}
	p := m.cache.PreferredPkgPath(projectRoot, url, version)
	_, err = DownloadGit(ctx, p, url, version, hash, m.ui)

	event := &tracking.TrackingEvent{
		Category: "pkg",
		Action:   "download-git",
		Fields: map[string]string{
			"url":     url,
			"version": version,
			"hash":    hash,
		},
	}
	if err != nil {
		event.Label = "failure"
	}
	m.track(ctx, event)

	return err
}

func (m *ProjectPkgManager) downloadLockFilePackages(ctx context.Context, lf *LockFile) error {
	encounteredError := false
	for pkgID, pe := range lf.Packages {
		if pe.Path == "" {
			if err := m.download(ctx, pe.URL, pe.Version, pe.Hash); err != nil {
				return err
			}
			continue
		}
		// Just check that the path is actually there and is a directory.
		isDir, err := isDirectory(pe.Path)
		if !isDir {
			m.ui.ReportError("Target of '%s' not a directory: '%s'", pkgID, pe.Path)
			encounteredError = true
		}
		if err != nil {
			return err
		}
	}
	if encounteredError {
		return ErrAlreadyReported
	}
	return nil
}

// prepareInstallGit prepares the installation of a git package.
// It finds the description of the package that should be installed.
// Returns the suggested prefix, the url, and the version
func (m *ProjectPkgManager) prepareInstallGit(ctx context.Context, pkgName string) (*Desc, error) {
	if pkgName == "" {
		return nil, m.ui.ReportError("Missing package name")
	}

	var versionStr *string = nil
	if atPos := strings.LastIndexByte(pkgName, '@'); atPos > 0 {
		v := pkgName[atPos+1:]
		versionStr = &v
		pkgName = pkgName[:atPos]
	}

	// Always search for shortened URLs.
	found, err := m.registries.searchShortURL(pkgName)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(pkgName, "/") {
		// Also search for the name.
		foundNames, err := m.registries.SearchName(pkgName)
		if err != nil {
			return nil, err
		}
		found = append(found, foundNames...)
	}

	if len(found) == 0 {
		return nil, m.ui.ReportError("Package '%s' not found", pkgName)
	}

	if versionStr == nil {
		found, err = found.WithoutLowerVersions(nil)
		if err != nil {
			return nil, err
		}
	} else {
		if *versionStr == "" {
			return nil, m.ui.ReportError("Missing version after '@' in '%s@'", pkgName)
		}
		constraints, err := parseInstallConstraint(*versionStr)
		if err != nil {
			return nil, m.ui.ReportError("Invalid version: '%s'", *versionStr)
		}
		found, err = found.WithoutLowerVersions(constraints)
		if err != nil {
			return nil, err
		}
	}
	var desc *Desc

	if len(found) == 0 {
		return nil, m.ui.ReportError("Package '%s-%s' not found", pkgName, versionStr)
	} else if len(found) > 1 {
		// Make one last attempt: if there is a package with the exact URL match, then we ignore
		// the other packages. In theory someone could have a bad name (although registries should
		// not accept them), or a URL could end with a full URL. For example: attack.com/github.com/real_package
		foundFullMatch := false
		for _, descReg := range found {
			if descReg.Desc.URL == pkgName {
				desc = descReg.Desc
				foundFullMatch = true
				break
			}
		}
		if !foundFullMatch {
			// TODO(florian): print all matching packages.
			return nil, m.ui.ReportError("More than one matching package '%s' found", pkgName)
		}
	} else {
		desc = found[0].Desc
	}

	return desc, nil
}

func (m *ProjectPkgManager) readSpecAndLock() (*Spec, *LockFile, error) {
	lfPath := m.Paths.LockFile
	lfExists, err := isFile(lfPath)
	if err != nil {
		return nil, nil, err
	}

	specPath := m.Paths.SpecFile

	specExists, err := isFile(specPath)
	if err != nil {
		return nil, nil, err
	}

	var spec *Spec
	if specExists {
		spec, err = ReadSpec(specPath, m.ui)
		if err != nil {
			return nil, nil, err
		}
	}

	var lf *LockFile
	if lfExists {
		lf, err = ReadLockFile(lfPath)
		if err != nil {
			return nil, nil, err
		}
	}

	if lfExists && specExists {
		// Do a check to ensure that the lockfile is correct. We don't want

		// to overwrite/discard the lockfile if someone just creates an empty
		// spec file.

		missingPrefixes := []string{}
		for prefix := range lf.Prefixes {
			if _, ok := spec.Deps[prefix]; !ok {
				missingPrefixes = append(missingPrefixes, prefix)
			}
		}
		if len(missingPrefixes) == 1 {
			return nil, nil, m.ui.ReportError("Lock file has prefix that isn't in package.yaml: '%s'", missingPrefixes[0])
		} else if len(missingPrefixes) > 1 {
			sort.Strings(missingPrefixes)
			return nil, nil, m.ui.ReportError("Lock file has prefixes that aren't in package.yaml: %s", strings.Join(missingPrefixes, ", "))
		}
	}

	if !specExists {
		if lfExists {
			spec, err = NewSpecFromLockFile(lf)
			if err != nil {
				return nil, nil, err
			}
		} else {
			spec = newSpec(specPath)
		}
	}
	return spec, lf, nil
}

func (m *ProjectPkgManager) writeSpecAndLock(spec *Spec, lf *LockFile) error {
	err := spec.WriteToFile()
	if err != nil {
		return err
	}

	return lf.WriteToFile()
}

// InstallPkg install the package identified by its identifier id.
// The id can be a path (when `isLocal` is true), a (suffix of a) package URL, or
// a package name. For non-local packages, the identifier can also be suffixed by
// a `@` followed by a version.
// When provided, the package is installed with the given prefix. Otherwise, the
// packages name (extracted from the description) is used.
// Returns (prefix, package-string, err).
func (m *ProjectPkgManager) InstallPkg(ctx context.Context, isLocal bool, prefix string, id string) (string, string, error) {

	var suggestedPrefix string
	var url string
	var concreteVersion string
	var versionConstraint string
	preferred := []versionedURL{}

	if isLocal {
		var err error
		suggestedPrefix, err = m.prepareInstallLocal(id)
		if err != nil {
			return "", "", err
		}
	} else {
		desc, err := m.prepareInstallGit(ctx, id)
		if err != nil {
			return "", "", err
		}
		suggestedPrefix = desc.Name
		url = desc.URL
		concreteVersion = desc.Version
		// The installation process automatically adjusts the version constraint of
		// installed packages to accept semver compatible versions.
		versionConstraint = "^" + concreteVersion
		id = ""
		preferred = append(preferred, versionedURL{
			URL:     url,
			Version: concreteVersion,
		})
	}

	if prefix == "" {
		prefix = suggestedPrefix
	}

	spec, lf, err := m.readSpecAndLock()
	if err != nil {
		return "", "", err
	}

	// Add the new dependency to the spec.
	err = spec.addDep(prefix, url, versionConstraint, id, m.ui)
	if err != nil {
		return "", "", err
	}

	updatedLock, err := m.downloadAndUpdateLock(ctx, spec, lf, preferred)
	if err != nil {
		return "", "", err
	}

	err = m.writeSpecAndLock(spec, updatedLock)
	if err != nil {
		return "", "", err
	}

	pkgString := id
	if id == "" {
		pkgString = url + "@" + concreteVersion
	}

	return prefix, pkgString, nil
}

func (m *ProjectPkgManager) Uninstall(ctx context.Context, prefix string) error {
	spec, lf, err := m.readSpecAndLock()
	if err != nil {
		return err
	}
	if _, ok := spec.Deps[prefix]; !ok {
		m.ui.ReportInfo("Prefix '%s' does not exist", prefix)
		return nil
	}
	delete(spec.Deps, prefix)

	updatedLock, err := m.downloadAndUpdateLock(ctx, spec, lf, nil)
	if err != nil {
		return err
	}
	return m.writeSpecAndLock(spec, updatedLock)
}

// Install downloads all dependencies.
// Simply downloads all dependencies, if forceRecompute is false, and a lock file
// without local dependencies exists.
// Otherwise (re)computes the lockfile, giving preference to versions that are
// listed in the lockfile (if it exists).
func (m *ProjectPkgManager) Install(ctx context.Context, forceRecompute bool) error {
	spec, lf, err := m.readSpecAndLock()
	if err != nil {
		return err
	}

	if forceRecompute || lf == nil {
		return m.update(ctx, spec, lf, true)
	}
	for _, pkg := range lf.Packages {
		if pkg.Path != "" {
			// Path dependencies might have changed constraints.
			// Recompute the dependencies, preferring the existing entries.
			return m.update(ctx, spec, lf, true)
		}
	}

	return m.downloadLockFilePackages(ctx, lf)
}

func (m *ProjectPkgManager) Update(ctx context.Context) error {
	spec, lf, err := m.readSpecAndLock()
	if err != nil {
		return err
	}

	return m.update(ctx, spec, lf, false)
}

func (m *ProjectPkgManager) update(ctx context.Context, spec *Spec, lf *LockFile, preferLock bool) error {
	preferredLock := &LockFile{}
	if preferLock {
		preferredLock = lf
	}
	updatedLock, err := m.downloadAndUpdateLock(ctx, spec, preferredLock, nil)
	if err != nil {
		return err
	}

	if lf != nil && lf.path != updatedLock.path {
		log.Fatal("Updated lock file '" + updatedLock.path + "' has different path than original '" + lf.path + "'")
	}

	return m.writeSpecAndLock(spec, updatedLock)
}

// downloadAndUpdateLock takes the current spec file and downloads all dependencies.
// It uses the old lockfile as hints for which package versions are preferred.
// Returns a lock-file corresponding to the resolved packages of the spec.
func (m *ProjectPkgManager) downloadAndUpdateLock(ctx context.Context, spec *Spec, oldLock *LockFile, preferred []versionedURL) (*LockFile, error) {
	solverDeps, err := spec.BuildSolverDeps(m.ui)
	if err != nil {
		return nil, err
	}
	solver, err := NewSolver(m.registries, m.ui)
	if err != nil {
		return nil, err
	}
	if oldLock != nil {
		for _, lockPkg := range oldLock.Packages {
			if lockPkg.URL != "" {
				preferred = append(preferred, versionedURL{
					URL:     lockPkg.URL,
					Version: lockPkg.Version,
				})
			}
		}
	}
	solver.SetPreferred(preferred)
	solution, err := solver.Solve(solverDeps)
	if err != nil {
		return nil, err
	}
	// Note that we need the downloaded packages, as we need their spec files to build
	// the updated lock file. Otherwise we don't have the prefixes of the packages.
	for url, versions := range solution {
		for version := range versions {
			// If we can't find the hash in the registries, we just use the empty string.
			hash, _ := m.registries.hashFor(url, version)
			err := m.download(ctx, url, version, hash)
			if err != nil {
				return nil, err
			}
		}
	}
	updatedLock, err := spec.BuildLockFile(solution, m.cache, m.registries, m.ui)
	if err != nil {
		return nil, err
	}
	updatedLock.optimizePkgIDs()
	return updatedLock, nil
}

// CleanPackages removes unused downloaded packages from the local cache.
func (m *ProjectPkgManager) CleanPackages() error {
	_, lf, err := m.readSpecAndLock()
	if err != nil {
		return err
	}
	if lf == nil {
		lf = &LockFile{}
	}

	rootPath := m.Paths.ProjectRootPath
	fullProjectPkgsPath, err := filepath.Abs(filepath.Join(rootPath, ProjectPackagesPath))
	if err != nil {
		return err
	}
	stat, err := os.Stat(fullProjectPkgsPath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	} else if !stat.IsDir() {
		return m.ui.ReportError("Packages cache path not a directory: '%s'", fullProjectPkgsPath)
	}

	// Build up a tree of segments so we can more efficiently
	// true: this is a full path, and no nested files must be removed.
	// false: this is a path that needs to be keep, but we must recurse.
	toKeep := map[string]bool{}
	for _, pkg := range lf.Packages {
		pkgPath, err := m.cache.FindPkg(rootPath, pkg.URL, pkg.Version)
		if err != nil {
			return err
		}
		if pkgPath != "" {
			fullPkgPath, err := filepath.Abs(pkgPath)
			if err != nil {
				return err
			}
			if strings.HasPrefix(fullPkgPath, fullProjectPkgsPath) {
				rel, err := filepath.Rel(fullProjectPkgsPath, fullPkgPath)
				if err != nil {
					return err
				}
				segments := strings.Split(rel, string(filepath.Separator))
				accumulated := ""
				for _, segment := range segments {
					if accumulated == "" {
						accumulated = segment
					} else {
						accumulated = filepath.Join(accumulated, segment)
					}
					toKeep[accumulated] = false
				}
				toKeep[accumulated] = true
			}
		}
	}
	// We now have all the project paths we want to keep.
	// Also add the README.md, that comes from the package manager.
	toKeep["README.md"] = false

	// Run through the cache directory and remove all the ones we don't need anymore.
	return filepath.Walk(fullProjectPkgsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == fullProjectPkgsPath {
			return nil
		}
		rel, err := filepath.Rel(fullProjectPkgsPath, path)
		if err != nil {
			return err
		}
		isFullPkgPath, ok := toKeep[rel]
		if !ok {
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if isFullPkgPath {
			return filepath.SkipDir
		}
		return nil
	})
}

// setupPaths sets the spec and lock file, searching in the given directory.
//
// Does not overwrite a set value (m.SpecFile or m.LockFile).
// If the given directory is empty, starts the search in the current working directory.
// If a file doesn't exists, returns the path for it in the given directory.
func NewProjectPaths(projectRoot string, lockPath string, specPath string) (*ProjectPaths, error) {

	if projectRoot != "" {
		if lockPath == "" {
			lockPath = lockPathForDir(projectRoot)
		}
		if specPath == "" {
			specPath = pkgPathForDir(projectRoot)
		}
		return &ProjectPaths{
			ProjectRootPath: projectRoot,
			LockFile:        lockPath,
			SpecFile:        specPath,
		}, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	startDir := dir
	lockPathCandidate := ""
	specPathCandidate := ""
	for {
		lockPathCandidate = lockPathForDir(dir)
		specPathCandidate = pkgPathForDir(dir)

		if info, err := os.Stat(lockPathCandidate); err == nil && !info.IsDir() {
			// Found the project root.
			break
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		if info, err := os.Stat(specPathCandidate); err == nil && !info.IsDir() {
			// Found the project root.
			break
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		// Prepare for the next iteration.
		// If there isn't one, we assume that the lock file and the spec file should
		// be in the starting directory.
		newDir := filepath.Dir(dir)
		if newDir == dir {
			dir = startDir
			lockPathCandidate = lockPathForDir(startDir)
			specPathCandidate = pkgPathForDir(startDir)
			break

		} else {
			dir = newDir
		}
	}

	if lockPath == "" {
		lockPath = lockPathCandidate
	}
	if specPath == "" {
		specPath = specPathCandidate
	}
	return &ProjectPaths{
		ProjectRootPath: dir,
		LockFile:        lockPath,
		SpecFile:        specPath,
	}, nil
}

// WithoutLowerVersions discards descriptions of packages where a higher
// version exists.
// If a constraint is given, then descriptions are first filtered out according to
// the constraint.
func (descs DescRegistries) WithoutLowerVersions(constraint version.Constraints) (DescRegistries, error) {
	if len(descs) == 0 {
		return descs, nil
	}

	filtered := DescRegistries{}
	if constraint == nil {
		filtered = descs
	} else {
		for _, desc := range descs {
			v, err := version.NewVersion(desc.Desc.Version)
			if err != nil {
				return nil, err
			}
			if constraint.Check(v) {
				filtered = append(filtered, desc)
			}
		}
	}

	if len(filtered) == 0 {
		return filtered, nil
	}

	sort.SliceStable(filtered, func(p, q int) bool {
		a := filtered[p]
		b := filtered[q]
		return a.Desc.IDCompare(b.Desc) < 0
	})
	// Only keep the highest version of a package.
	to := 0
	for i := 1; i < len(filtered); i++ {
		current := filtered[i]
		previous := filtered[i-1]
		if current.Desc.URL == previous.Desc.URL {
			// Same package. Maybe different version, but the latter is either higher or equal.
			filtered[to] = current
		} else {
			to++
			filtered[to] = current
		}
	}
	filtered = filtered[0 : to+1]
	return filtered, nil
}
