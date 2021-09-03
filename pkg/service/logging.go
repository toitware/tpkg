package service

import (
	"fmt"

	"github.com/toitware/tpkg.git/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func fxLogger() (fx.Printer, error) {
	// setup output logger
	l, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return zap.NewStdLog(l), nil
}

func ensureFxLogger(printer fx.Printer, err error) fx.Printer {
	if err != nil {
		panic(err)
	}
	return printer
}

func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()

	var options []zap.Option
	switch cfg.Logging.Backend {
	case "":
		zapCfg = zap.NewDevelopmentConfig()
	case "humio":
		// Nothing, use production out of the box.
	default:
		return nil, fmt.Errorf("unknown logging backend: '%s'", cfg.Logging.Backend)
	}

	if level := cfg.Logging.Level; level != "" {
		if err := zapCfg.Level.UnmarshalText([]byte(level)); err != nil {
			return nil, err
		}
	}

	zapCfg.InitialFields = map[string]interface{}{}
	for k, v := range cfg.Logging.Tags {
		zapCfg.InitialFields[k] = v
	}

	logger, err := zapCfg.Build(options...)
	if err != nil {
		return nil, err
	}

	logger.Info("started")

	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)

	return logger, nil
}
