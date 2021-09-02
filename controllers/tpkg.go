package controllers

import (
	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tracking"
)

func provideTpkgRegistry(cfg *config.Config, cache tpkg.Cache) (tpkg.Registry, error) {
	return tpkg.NewGitRegistry("registry", cfg.Registry.Url, cache)
}

func provideManager(registry tpkg.Registry, cache tpkg.Cache, ui tpkg.UI) *tpkg.Manager {
	return tpkg.NewManager(tpkg.Registries{registry}, cache, ui, tracking.NopTrack)
}
