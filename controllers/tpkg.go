package controllers

import (
	"github.com/toitware/tpkg.git/config"
	"github.com/toitware/tpkg.git/pkg/tpkg"
	"github.com/toitware/tpkg.git/pkg/tracking"
)

func provideTpkgRegistry(cfg *config.Config, cache tpkg.Cache) (tpkg.Registry, error) {
	return tpkg.NewGitRegistry("registry", cfg.Registry.Url, cache)
}

func provideManager(registry tpkg.Registry, cache tpkg.Cache, ui tpkg.UI) *tpkg.Manager {
	return tpkg.NewManager(tpkg.Registries{registry}, cache, ui, tracking.NopTrack)
}
