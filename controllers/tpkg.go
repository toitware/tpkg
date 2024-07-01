// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package controllers

import (
	"github.com/toitlang/tpkg/pkg/tpkg"
	"github.com/toitlang/tpkg/pkg/tracking"
	"github.com/toitware/tpkg/config"
)

func provideTpkgRegistry(cfg *config.Config, cache tpkg.Cache) (tpkg.Registry, error) {
	if err := populateSSHKeyFile(cfg); err != nil {
		return nil, err
	}

	return tpkg.NewSSHGitRegistry("registry", cfg.Registry.Url, cache, cfg.Registry.SSHKeyFile, cfg.Registry.Branch)
}

func provideManager(registry tpkg.Registry, cache tpkg.Cache, ui tpkg.UI) *tpkg.Manager {
	return tpkg.NewManager(tpkg.Registries{registry}, cache, nil, ui, tracking.NopTrack)
}
