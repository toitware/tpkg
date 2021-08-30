package controllers

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		provideRegistry,
		provideTpkgRegistry,
		provideToitdoc,
		provideManager,
	),
	fx.Invoke(
		initRegistry,
	),
)
