package service

import (
	"github.com/toitware/toit.git/tools/tpkg/pkg/service/debug"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		provideLogger,
		provideTally,
		provideReporter,
		fxLogger,
	),
	fx.Invoke(
		loadTally,
	),
	debug.Module,
	fx.Logger(ensureFxLogger(fxLogger())),
)
