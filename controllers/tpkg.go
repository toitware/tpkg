package controllers

import (
	"github.com/toitware/tpkg/config"
	"github.com/toitware/tpkg/pkg/tpkg"
	"github.com/toitware/tpkg/pkg/tracking"
)

func provideTpkgRegistry(cfg *config.Config, cache tpkg.Cache) (tpkg.Registry, error) {
	return tpkg.NewSSHGitRegistry("registry", cfg.Registry.Url, cache, cfg.Registry.SSHKeyFile, cfg.Registry.Branch)
}

func provideManager(registry tpkg.Registry, cache tpkg.Cache, ui tpkg.UI) *tpkg.Manager {
	return tpkg.NewManager(tpkg.Registries{registry}, cache, ui, tracking.NopTrack)
}
