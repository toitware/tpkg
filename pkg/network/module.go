package network

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		provideGRPCServer,
		provideMux,
		provideHTTPServer,
		provideRouter,
		provideHostAddress,
	),
	fx.Invoke(
		bindHTTPServer,
	),
)
