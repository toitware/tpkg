// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"time"

	"github.com/toitware/tpkg/config"
	"github.com/toitware/tpkg/pkg/service/debug"
	"github.com/uber-go/tally"
	"github.com/uber-go/tally/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func provideReporter(cfg *config.Config, logger *zap.Logger, mux debug.Mux) (tally.CachedStatsReporter, error) {
	if !cfg.Metrics.Enabled {
		return nil, nil
	}

	r := prometheus.NewReporter(prometheus.Options{})
	mux.Handle("/metrics", r.HTTPHandler())
	return r, nil
}

func provideTally(lc fx.Lifecycle, reporter tally.CachedStatsReporter, cfg *config.Config) (tally.Scope, error) {
	if !cfg.Metrics.Enabled {
		return tally.NoopScope, nil
	}

	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Tags:           cfg.Metrics.Tags,
		Prefix:         cfg.Metrics.Prefix,
		CachedReporter: reporter,
		Separator:      prometheus.DefaultSeparator,
	}, time.Second)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return closer.Close()
		},
		OnStart: func(ctx context.Context) error {
			return nil
		},
	})

	scope.Counter("boot").Inc(1)

	return scope, nil
}

func loadTally(scope tally.Scope) {

}
