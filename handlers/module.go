package handlers

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		provideCache,
		provideRegistryService,
		provideLoggerUI,
		provideHTTPHandlers,
	),
	fx.Invoke(
		bindRegistryService,
		bindHTTPHandlers,
		proxyHTTPToGRPC,
	),
)
