// Copyright (C) 2023 Toitware ApS.

package controllers

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/toitlang/tpkg/pkg/tpkg"
	"github.com/toitware/tpkg/config"
	"go.uber.org/zap"
)

func createFileRegistry(t *testing.T) string {
	dir, err := ioutil.TempDir("", "tmp")
	require.NoError(t, err)

	r, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	// Add a README file to the git repository, so it's not empty.
	err = ioutil.WriteFile(filepath.Join(dir, "README.md"), []byte("Hello World"), 0644)
	require.NoError(t, err)

	// Commit the README file.
	w, err := r.Worktree()
	require.NoError(t, err)

	_, err = w.Add("README.md")
	require.NoError(t, err)

	_, err = w.Commit("Initial commit", &git.CommitOptions{})
	require.NoError(t, err)

	// Create a non-master branch.
	// This is so we can commit to master without having it checked out.
	branchName := plumbing.NewBranchReferenceName("testing")
	headRef, err := r.Head()
	require.NoError(t, err)
	ref := plumbing.NewHashReference(branchName, headRef.Hash())
	err = r.Storer.SetReference(ref)
	require.NoError(t, err)

	// Check it out.
	err = w.Checkout(&git.CheckoutOptions{
		Branch: branchName,
	})
	require.NoError(t, err)

	return dir
}

func checkPkgExists(t *testing.T, registry *registry, pkg, version string) {
	dir := registry.remoteRegistryConfig.Url

	// Check out the master branch.
	r, err := git.PlainOpen(dir)
	require.NoError(t, err)

	w, err := r.Worktree()
	require.NoError(t, err)

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/master"),
	})
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "packages", pkg, version, tpkg.DescriptionFileName))
	require.NoError(t, err)
}

func withRegistry(t *testing.T, f func(context.Context, *registry)) {
	ctx := context.Background()

	remoteRegistryPath := createFileRegistry(t)
	defer os.RemoveAll(remoteRegistryPath)

	remoteRegistryConfig := config.Registry{
		Url:    remoteRegistryPath,
		Branch: "master",
	}

	ui := tpkg.FmtUI

	cacheDir, err := ioutil.TempDir("", "tmp")
	require.NoError(t, err)

	cache := tpkg.NewCache(cacheDir, ui)
	remoteRegistry, err := tpkg.NewGitRegistry("testing", remoteRegistryPath, cache)
	require.NoError(t, err)

	logger := zap.NewNop()

	registry := &registry{
		logger:               logger,
		lookup:               map[string]*Package{},
		packages:             []*Package{},
		remoteRegistry:       remoteRegistry,
		remoteRegistryConfig: remoteRegistryConfig,
		cache:                cache,
		ui:                   ui,
	}

	f(ctx, registry)
}

func Test_register(t *testing.T) {
	withRegistry(t, func(ctx context.Context, registry *registry) {
		err := registry.RegisterPackage(ctx, "github.com/toitware/toit-morse", "v1.0.6")
		assert.NoError(t, err)

		// Expect the file to be committed to the remote registry.
		checkPkgExists(t, registry, "github.com/toitware/toit-morse", "1.0.6")
	})
}

func Test_registerHttps(t *testing.T) {
	withRegistry(t, func(ctx context.Context, registry *registry) {
		err := registry.RegisterPackage(ctx, "https://github.com/toitware/toit-morse", "v1.0.6")
		assert.NoError(t, err)

		// Expect the file to be committed to the remote registry.
		checkPkgExists(t, registry, "github.com/toitware/toit-morse", "1.0.6")
	})
}

func Test_registerNoName(t *testing.T) {
	// Test that it is an error now if the package doesn't have a name/description in the
	// package.yaml file.
	withRegistry(t, func(ctx context.Context, registry *registry) {
		err := registry.RegisterPackage(ctx, "github.com/toitware/toit-morse", "v1.0.0")
		assert.Error(t, err)
	})
}
