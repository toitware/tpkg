package main

import (
	"github.com/toitware/tpkg.git/config"
	"github.com/toitware/tpkg.git/controllers"
	"github.com/toitware/tpkg.git/handlers"
	"github.com/toitware/tpkg.git/pkg/network"
	"github.com/toitware/tpkg.git/pkg/service"
	"github.com/toitware/tpkg.git/pkg/toitdoc"
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
