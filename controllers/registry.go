package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"go.uber.org/fx"
	"go.uber.org/ratelimit"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func provideRegistry(config *config.Config, cache tpkg.Cache, logger *zap.Logger, ui tpkg.UI) (*registry, Registry, error) {
	r, err := tpkg.NewSSHGitRegistry("registry", config.Registry.Url, cache, config.Registry.SSHKeyFile, config.Registry.Branch)
	if err != nil {
		return nil, nil, err
	}

	if _, err := os.Stat(config.Registry.SSHKeyFile); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("Failed to load SSH key from path: '%s'", config.Registry.SSHKeyFile)
	}

	authMethod, err := ssh.NewPublicKeysFromFile("git", config.Registry.SSHKeyFile, "")
	if err != nil {
		return nil, nil, err
	}
	res := &registry{
		logger:               logger,
		lookup:               map[string]*Package{},
		packages:             []*Package{},
		remoteRegistry:       r,
		remoteRegistryConfig: config.Registry,
		authMethod:           authMethod,
		cache:                cache,
		syncLimit:            ratelimit.New(1, ratelimit.Per(5*time.Second), ratelimit.WithoutSlack),
		ui:                   ui,
	}

	return res, res, nil
}

func initRegistry(lc fx.Lifecycle, registry *registry) {
	lc.Append((fx.Hook{
		OnStart: func(ctx context.Context) error {
			return registry.sync(ctx)
		},
	}))
}

type Registry interface {
	Packages(ctx context.Context) ([]*Package, error)
	Package(ctx context.Context, url string) (*Package, error)
	Sync(ctx context.Context) error
	RegisterPackage(ctx context.Context, url string, version string) error
}

type Package struct {
	Lookup       map[string]*tpkg.Desc
	Descriptions []*tpkg.Desc // Descriptions sorted by semver.
}

func (p *Package) Latest() *tpkg.Desc {
	return p.Descriptions[len(p.Descriptions)-1]
}

type registry struct {
	lookup   map[string]*Package
	packages []*Package // Packages sorted by name.

	logger               *zap.Logger
	remoteRegistry       tpkg.Registry
	remoteRegistryConfig config.Registry
	authMethod           transport.AuthMethod
	cache                tpkg.Cache
	ui                   tpkg.UI
	syncLimit            ratelimit.Limiter
	syncMutex            sync.Mutex
}

func (r *registry) Packages(ctx context.Context) ([]*Package, error) {
	r.syncMutex.Lock()
	defer r.syncMutex.Unlock()
	return r.packages, nil
}

func (r *registry) Package(ctx context.Context, url string) (*Package, error) {
	r.syncMutex.Lock()
	defer r.syncMutex.Unlock()
	p, ok := r.lookup[url]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "package '%s' did not exist", url)
	}
	return p, nil
}

func (r *registry) Sync(ctx context.Context) error {
	r.syncLimit.Take()
	return r.sync(ctx)
}

func (r *registry) sync(ctx context.Context) error {
	if err := r.remoteRegistry.Load(ctx, true, r.cache, r.ui); err != nil {
		return err
	}
	entries := r.remoteRegistry.Entries()
	packages, packagesLookup := buildPackageStructure(entries)

	r.syncMutex.Lock()
	defer r.syncMutex.Unlock()
	r.packages = packages
	r.lookup = packagesLookup
	return nil
}

func buildPackageStructure(entries []*tpkg.Desc) ([]*Package, map[string]*Package) {
	packagesLookup := map[string]*Package{}
	packages := []*Package{}

	for _, e := range entries {
		if _, ok := packagesLookup[e.URL]; !ok {
			pkg := &Package{
				Lookup:       map[string]*tpkg.Desc{},
				Descriptions: []*tpkg.Desc{},
			}
			packagesLookup[e.URL] = pkg
			packages = append(packages, pkg)
		}
		pkg := packagesLookup[e.URL]
		pkg.Descriptions = append(pkg.Descriptions, e)
		pkg.Lookup[e.Version] = e
	}

	for _, p := range packages {
		sort.Slice(p.Descriptions, func(i, j int) bool {
			return p.Descriptions[i].IDCompare(p.Descriptions[j]) < 0
		})
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Descriptions[0].IDCompare(packages[j].Descriptions[0]) < 0
	})
	return packages, packagesLookup
}

func (r *registry) RegisterPackage(ctx context.Context, url string, version string) error {

	desc, err := tpkg.ScrapeDescriptionGit(ctx, url, version, tpkg.DisallowLocalDeps, false, r.ui)
	if err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "tmp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	registryUrl := r.remoteRegistryConfig.Url
	if !filepath.IsAbs(registryUrl) {
		registryUrl = "ssh://" + registryUrl
	}
	branch := r.remoteRegistryConfig.Branch
	repository, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:           registryUrl,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Auth:          r.authMethod,
	})
	if err != nil {
		return err
	}

	descPath, err := desc.WriteInDir(dir)
	if err != nil {
		return err
	}

	relDescPath, err := filepath.Rel(dir, descPath)
	if err != nil {
		return err
	}

	wt, err := repository.Worktree()
	if err != nil {
		return err
	}

	if err := wt.AddWithOptions(&git.AddOptions{Path: relDescPath}); err != nil {
		return err
	}

	if _, err := wt.Commit(fmt.Sprintf("Add %s version %s", url, version), &git.CommitOptions{
		Author: &object.Signature{
			Name: "Toit package registry",
			When: time.Now(),
		},
	}); err != nil {
		return err
	}

	if err := repository.Push(&git.PushOptions{Auth: r.authMethod}); err != nil {
		return err
	}

	return nil
}
