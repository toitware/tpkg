// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	"github.com/toitware/tpkg/pkg/git"
)

type Registries []Registry

// Registry is a source of package descriptions.
type Registry interface {
	// Name of the registry.
	Name() string
	// Loads the registry into memory.
	// Synchronizes the registry first, if 'sync' is true.
	// Synchronization installs the registry first if necessary. It then
	// downloads the latest packages.
	Load(ctx context.Context, sync bool, cache Cache, ui UI) error
	// Describes this registry. Used when showing where a specification comes from.
	Describe() string
	// All the loaded entries. If the registry hasn't been loaded yet returns nil.
	Entries() []*Desc
	// Searches for the given package name in the registry.
	// Returns all matching packages.
	SearchName(name string) ([]*Desc, error)
	// Searches for needle.
	// The search uses all description information (including description, authors, ...)
	// to find the package.
	SearchAll(needle string) ([]*Desc, error)
	// Searches for a package with the given URL and version.
	SearchURL(url string, version string) ([]*Desc, error)
	// searchShortURL searches for the given 'shortened' parameter.
	// Either shortened must be equal to the URL, or it must be a suffix of it, so
	// that the remaining URL ends with '/'.
	// For example `foo/bar` is a shortened URL of `github.com/foo/bar`, but not of
	// `github.com/XXfoo/bar`.
	searchShortURL(shortened string) ([]*Desc, error)
}

// RegistryConfig can be used to load a registry with
// LoadRegistry or LoadRegistries.
type RegistryConfig struct {
	Name string       `yaml:"name"`
	Kind RegistryKind `yaml:"kind"`
	Path string       `yaml:"path"`
}

type RegistryConfigs []RegistryConfig

// RegistryKind specifies how to load a registry.
// See PathKind.
type RegistryKind string

const (
	// RegistryKindLocal specifies that the corresponding registry should treated like
	// a simple folder with descriptions in it.
	RegistryKindLocal RegistryKind = "local"
	// RegistryKindGit specifies that the registry is backed by a git-repository.
	RegistryKindGit RegistryKind = "git"
)

// IsValid returns whether the registry kind is valid. The kind value should be one
// of the exported kinds. See PathKind.
func (k RegistryKind) IsValid() bool {
	return k == RegistryKindLocal || k == RegistryKindGit
}

// Load loads the registry given by its configuration.
func (cfg RegistryConfig) Load(ctx context.Context, sync bool, cache Cache, ui UI) (Registry, error) {
	if !cfg.Kind.IsValid() {
		err := ui.ReportError("Unexpected registry config %v", cfg.Kind)
		return nil, err
	}
	var registry Registry
	if cfg.Kind == RegistryKindLocal {
		registry = NewLocalRegistry(cfg.Name, cfg.Path)
	} else {
		var err error
		registry, err = NewGitRegistry(cfg.Name, cfg.Path, cache)
		if err != nil {
			return nil, err
		}
	}
	if err := registry.Load(ctx, sync, cache, ui); err != nil {
		return nil, err
	}
	return registry, nil
}

// Load takes the registry configuration and loads the
// corresponding registries into memory.
func (configs RegistryConfigs) Load(ctx context.Context, sync bool, cache Cache, ui UI) (Registries, error) {
	result := []Registry{}
	for _, config := range configs {
		registry, err := config.Load(ctx, sync, cache, ui)
		if err != nil {
			return nil, err
		}
		result = append(result, registry)
	}
	return result, nil
}

type pathRegistry struct {
	name    string
	path    string
	entries []*Desc
}

type gitRegistry struct {
	pathRegistry
	url string
}

var (
	_ Registry = (*pathRegistry)(nil)
	_ Registry = (*gitRegistry)(nil)
)

func (p *pathRegistry) Name() string {
	return p.name
}

func (p *pathRegistry) Describe() string {
	// For now just use the name and path as description.
	if p.name == "" {
		return p.path
	}
	return fmt.Sprintf("%s: %s", p.name, p.path)
}

var blocklist = []glob.Glob{
	glob.MustCompile(".**", '/'), // Any hidden file or directory, including .git.
}

func (p *pathRegistry) Load(_ context.Context, sync bool, _ Cache, ui UI) error {
	entries := []*Desc{}
	err := filepath.Walk(p.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and folders.
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(p.path, path)
		if err != nil {
			return err
		}

		// The entry directory is never blocklisted.
		if rel == "." {
			return nil
		}

		for _, glob := range blocklist {
			if glob.Match(rel) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			return nil
		}

		e := filepath.Ext(rel)
		if e != ".yaml" && e != ".yml" {
			return nil
		}

		var entry Desc
		if err = entry.ParseFile(path, ui); err != nil {
			return err
		}
		entries = append(entries, &entry)
		return nil
	})
	if err != nil {
		return err
	}
	p.entries = entries
	return nil
}

func (p *pathRegistry) Entries() []*Desc {
	return p.entries
}

// NewLocalRegistry creates a new path registry.
// Path registries simply find all package descriptions in a certain path.
func NewLocalRegistry(name string, path string) Registry {
	return newLocalRegistry(name, path)
}

func newLocalRegistry(name string, path string) *pathRegistry {
	return &pathRegistry{
		name: name,
		path: path,
	}
}

func matchName(name string, desc *Desc) bool {
	// TODO(florian): take qualifying path into account.
	return name == desc.Name
}

func matchDescription(needle string, desc *Desc) bool {
	return strings.Contains(desc.Description, needle)
}

func matchURL(needle string, desc *Desc) bool {
	return strings.Contains(desc.URL, needle)
}

func (p *pathRegistry) SearchName(name string) ([]*Desc, error) {
	result := []*Desc{}
	for _, entry := range p.entries {
		if matchName(name, entry) {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (p *pathRegistry) SearchAll(needle string) ([]*Desc, error) {
	result := []*Desc{}
	for _, entry := range p.entries {
		if matchName(needle, entry) || matchDescription(needle, entry) || matchURL(needle, entry) {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (p *pathRegistry) SearchURL(url string, version string) ([]*Desc, error) {
	result := []*Desc{}
	for _, entry := range p.entries {
		if entry.URL == url && entry.Version == version {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (p *pathRegistry) searchShortURL(shortened string) ([]*Desc, error) {
	result := []*Desc{}
	withSlash := "/" + shortened
	for _, entry := range p.entries {
		if entry.URL == shortened || strings.HasSuffix(entry.URL, withSlash) {
			result = append(result, entry)
		}
	}
	return result, nil
}

// NewGitRegistry creates a new registry that is backed by a git-repository.
// The data is fetched (cloned) during 'Load' when 'sync' is true.
func NewGitRegistry(name string, url string, cache Cache) (Registry, error) {
	return newGitRegistry(name, url, cache)
}

func newGitRegistry(name string, url string, cache Cache) (*gitRegistry, error) {
	p, err := cache.FindRegistry(url)
	if err != nil {
		return nil, err
	}
	return &gitRegistry{
		pathRegistry: *newLocalRegistry(name, p),

		url: url,
	}, nil
}

func (gr *gitRegistry) Describe() string {
	return fmt.Sprintf("%s: %s", gr.name, gr.url)
}

func (gr *gitRegistry) Load(ctx context.Context, sync bool, cache Cache, ui UI) error {
	if sync {
		if gr.path == "" {

			p := cache.PreferredRegistryPath(gr.url)
			url := gr.url

			println("preferred path: ", p)
			for _, c := range p {
				if c == '\\' {
					print("@")
				} else {
					print(string(c))
				}
				print(" ")
			}
			println("")
			var err error
			println("Trying other out dir")
			out := filepath.Join(cache.pkgCachePaths[0], "t")
			for _, branch := range []string{"master", "main", "trunk"} {
				_, err = git.Clone(ctx, out, &git.CloneOptions{
					URL:          url,
					SingleBranch: true,
					Branch:       branch,
				})
				if err == nil {
					break
				}
			}
			if err != nil {
				println("Error while cloning into", out)
				return err
			}

			for _, branch := range []string{"master", "main", "trunk"} {
				_, err = git.Clone(ctx, p, &git.CloneOptions{
					URL:          url,
					SingleBranch: true,
					Branch:       branch,
				})
				if err == nil {
					break
				}
			}
			if err != nil {
				println("Error while cloning", err.Error())
				return err
			}
			gr.pathRegistry.path = p
		} else {
			err := git.Pull(gr.path)
			if err != nil {
				return err
			}

		}
	}
	if gr.path == "" {
		// The repository was never cloned. Don't try to load anything.
		return nil
	}
	return gr.pathRegistry.Load(ctx, sync, cache, ui)
}

func (registries Registries) searchInRegistries(searchFun func(Registry) ([]*Desc, error)) (DescRegistries, error) {
	result := []DescRegistry{}
	for _, registry := range registries {
		found, err := searchFun(registry)
		if err != nil {
			return nil, err
		}
		for _, desc := range found {
			result = append(result, DescRegistry{
				Desc:     desc,
				Registry: registry,
			})
		}
	}
	return result, nil
}

// SearchName searches for the given name in all registries.
func (registries Registries) SearchName(name string) (DescRegistries, error) {
	return registries.searchInRegistries(func(registry Registry) ([]*Desc, error) {
		return registry.SearchName(name)
	})
}

// SearchAll searches for the given needle in the names and descriptions of all registries.
func (registries Registries) SearchAll(needle string) (DescRegistries, error) {
	return registries.searchInRegistries(func(registry Registry) ([]*Desc, error) {
		return registry.SearchAll(needle)
	})
}

// SearchURL searches for the package with the given url and version in all registries.
func (registries Registries) searchURL(url string, version string) (DescRegistries, error) {
	return registries.searchInRegistries(func(registry Registry) ([]*Desc, error) {
		return registry.SearchURL(url, version)
	})
}

// SearchShortUrl searches for the shortened url in all registries.
func (registries Registries) searchShortURL(url string) (DescRegistries, error) {
	return registries.searchInRegistries(func(registry Registry) ([]*Desc, error) {
		return registry.searchShortURL(url)
	})
}

type sshGitRegistry struct {
	gitRegistry
	sshPath string
	branch  string
}

func NewSSHGitRegistry(name string, url string, cache Cache, sshPath string, branch string) (Registry, error) {
	registry, err := newGitRegistry(name, url, cache)
	if err != nil {
		return nil, err
	}
	return &sshGitRegistry{
		gitRegistry: *registry,
		sshPath:     sshPath,
		branch:      branch,
	}, nil
}

func (gr *sshGitRegistry) Load(ctx context.Context, sync bool, cache Cache, ui UI) error {
	if !sync {
		if gr.path == "" {
			return ui.ReportError("Registry '%s' not synced", gr.Name())
		}
	} else {

		if gr.path == "" {
			p := cache.PreferredRegistryPath(gr.url)
			_, err := git.Clone(ctx, p, &git.CloneOptions{
				URL:    gr.url,
				Branch: gr.branch,

				SingleBranch: true,
				SSHPath:      gr.sshPath,
			})
			if err != nil {
				return err
			}
			gr.pathRegistry.path = p
		} else {
			err := git.Pull(gr.path)
			if err != nil {
				return err
			}

			if err != nil {
				return err
			}

		}
	}
	return gr.pathRegistry.Load(ctx, sync, cache, ui)
}

// hashFor finds the has for the package with the given url and version.
func (registries Registries) hashFor(url string, version string) (string, error) {
	for _, registry := range registries {
		for _, entry := range registry.Entries() {
			if entry.URL == url && entry.Version == version {
				return entry.Hash, nil
			}
		}
	}
	return "", fmt.Errorf("not found")
}
