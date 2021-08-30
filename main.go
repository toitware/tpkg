package main

import (
	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/controllers"
	"github.com/toitware/toit.git/tools/tpkg/handlers"
	"github.com/toitware/toit.git/tools/tpkg/pkg/network"
	"github.com/toitware/toit.git/tools/tpkg/pkg/service"
	"github.com/toitware/toit.git/tools/tpkg/pkg/toitdoc"
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
