package toitdoc

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		provideManager,
		provideGenerator,
	),
)
