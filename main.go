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
