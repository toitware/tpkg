// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"github.com/toitware/tpkg/config"
	"github.com/toitware/tpkg/controllers"
	"github.com/toitware/tpkg/handlers"
	"github.com/toitware/tpkg/pkg/network"
	"github.com/toitware/tpkg/pkg/service"
	"github.com/toitware/tpkg/pkg/toitdoc"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module,
		handlers.Module,
		service.Module,
		network.Module,
		controllers.Module,
		toitdoc.Module,
	).Run()
}
