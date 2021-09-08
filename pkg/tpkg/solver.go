// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"fmt"
	"sort"

	"github.com/hashicorp/go-version"
	"github.com/toitware/tpkg/pkg/set"
)

// Solver is a simple constraint solver for the Toit package manager.
type Solver struct {
	db pkgDB
	ui UI
}

// pkgDB is a map from package-url to all the existing packages of that url.
// The solver tries the list of packages one by one to find the one that
// works. As such, the order of packages is important.
type pkgDB map[string][]solverPkg

// solverPkg represents a package in the solver.
// It only needs the version and the dependencies. (The name is stored as
// key in the map that contains all packages).
type solverPkg struct {
	version *version.Version
	deps    []SolverDep
}

// SolverDep represents a dependency for the solver.
// It needs the target's package name and the version constraints for it.
type SolverDep struct {
	url         string
	constraints version.Constraints
}

// Solution is a map from pkg-url to a set of versions.
type Solution map[string]set.String

// versionedURL combines a URL and a version.
type versionedURL struct {
	URL     string
	Version string
}

type byVersion []solverPkg

func (a byVersion) Len() int           { return len(a) }
func (a byVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byVersion) Less(i, j int) bool { return a[i].version.LessThan(a[j].version) }

func NewSolverDep(url string, constraintString string) (SolverDep, error) {
	constraints, err := parseConstraint(constraintString)
	if err != nil {
		return SolverDep{}, err
	}

	return SolverDep{
		url:         url,
		constraints: constraints,
	}, nil
}

func convertDeps(descDeps []descPackage) ([]SolverDep, error) {
	result := []SolverDep{}
	for _, dep := range descDeps {
		solverDep, err := NewSolverDep(dep.URL, dep.Version)
		if err != nil {
			return nil, err
		}
		result = append(result, solverDep)
	}
	return result, nil
}

func NewSolver(registries Registries, ui UI) (*Solver, error) {
	result := &Solver{
		db: map[string][]solverPkg{},
		ui: ui,
	}

	for _, reg := range registries {
		for _, desc := range reg.Entries() {
			v, err := version.NewVersion(desc.Version)
			if err != nil {
				return nil, err
			}
			deps, err := convertDeps(desc.Deps)
			if err != nil {
				return nil, err
			}
			pkgs := result.db[desc.URL]
			pkgs = append(pkgs, solverPkg{
				version: v,
				deps:    deps,
			})
			result.db[desc.URL] = pkgs
		}
	}
	// Sort entries.
	for _, pkgs := range result.db {
		sort.Sort(byVersion(pkgs))
		// Reverse the list, as we want higher versions first.
		// We could try to sort the pkgs already in opposite version order, but that
		// would require writing a different `byVersion` comparator, or wrapping it.
		// It's just easier and cleaner to reverse the list after having sorted it.
		for i, j := 0, len(pkgs)-1; i < j; i, j = i+1, j-1 {
			pkgs[i], pkgs[j] = pkgs[j], pkgs[i]
		}
	}
	return result, nil
}

// SetPreferred marks the list of versionedURLs as preferred.
func (s *Solver) SetPreferred(preferred []versionedURL) {
	// Start from the back, so that the given preferred versions are found in order.
	for i := len(preferred) - 1; i >= 0; i-- {
		versioned := preferred[i]
		url := versioned.URL
		version, err := version.NewVersion(versioned.Version)
		if err != nil {
			// Version didn't parse. Just skip it.
			continue
		}
		pkgs, ok := s.db[url]
		if !ok {
			// We don't have a package with that URL.
			continue
		}
		// In theory we could try to use binary search (since the packages are sorted by
		// version). However, we don't have a guarantee that we didn't already reorder the
		// packages with a preferred package.
		for j := 0; j < len(pkgs); j++ {
			pkg := pkgs[j]
			if pkg.version.Equal(version) {
				// Take the current pkg and move it to the first slot.
				for k := j; k > 0; k-- {
					pkgs[k] = pkgs[k-1]
				}
				pkgs[0] = pkg
				break
			}
		}
		// We might not have found the version, but that's fine. In that case we
		// didn't modify anything.
	}
}

func (s *Solver) solveDep(dep SolverDep, solution Solution) error {
	// For now just find the first package that satisfies the constraint.
	// Since the packages are sorted, we should find the package with the highest
	// version.
	url := dep.url
	available, ok := s.db[url]
	if !ok {
		return s.ui.ReportError("Package '%s' not found", url)
	}
	constraints := dep.constraints
	// TODO(florian): this now ends up being an O(n * m) operation, where
	// 'n' is the number of referenced packages, and 'm' is the number of versions
	// for the package. It's relatively easy to make this more efficient.
	// TODO(florian): we want to agree on a common package for minor versions.
	// Currently we just take the highest version that satisfies the constraint,
	// potentially leading to multiple versions of the same package that only differ
	// in the minor version.
	for _, pkg := range available {
		if constraints.Check(pkg.version) {
			// Found a valid version.
			versions, ok := solution[url]
			if !ok {
				versions = set.String{}
				solution[url] = versions
			}
			versionStr := pkg.version.String()
			if !versions.Contains(versionStr) {
				versions.Add(versionStr)
				err := s.solveDeps(pkg.deps, solution)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	// No package found.
	return s.ui.ReportError("No version of '%s' satisfying '%s'", url, constraints.String())
}

func (s *Solver) solveDeps(deps []SolverDep, solution Solution) error {
	for _, dep := range deps {
		err := s.solveDep(dep, solution)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Solver) Solve(deps []SolverDep) (Solution, error) {
	result := Solution{}
	err := s.solveDeps(deps, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// versionFor returns the concrete version of the package url with the given constraintsString.
func (sol Solution) versionFor(url string, constraintsString string, ui UI) (string, error) {
	versions, ok := sol[url]
	if !ok {
		return "", fmt.Errorf("package solution missing package '%s'", url)
	}
	constraints, err := parseConstraint(constraintsString)
	if err != nil {
		return "", err
	}
	// TODO(florian): we are parsing the version multiple times, and are running
	// through all existing versions multiple times. This can be optimized.
	for versionString := range versions {
		version, err := version.NewVersion(versionString)
		if err != nil {
			return "", err
		}
		if constraints.Check(version) {
			return versionString, nil
		}
	}
	return "", fmt.Errorf("package solution missing target for '%s' with constraint '%s'", url, constraintsString)
}
