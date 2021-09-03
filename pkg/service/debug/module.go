package debug

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/toitware/tpkg.git/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Options(
	fx.Provide(
		provideMux,
	),
	fx.Invoke(
		initDebugger,
	),
)

type Mux struct {
	*mux.Router
}

func provideMux() Mux {
	mux := mux.NewRouter()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	return Mux{mux}
}

func initDebugger(log *zap.Logger, lc fx.Lifecycle, cfg *config.Config, m Mux) {
	debugPort := cfg.DebugPort
	if debugPort == nil {
		return
	}

	srvr := &http.Server{
		Addr:    fmt.Sprintf(":%d", *debugPort),
		Handler: m,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				l, err := net.Listen("tcp4", srvr.Addr)
				if err != nil {
					log.Error("Failed to listen on debug port", zap.Error(err))
				}
				log.Info("Started debug server", zap.String("address", l.Addr().String()))
				if err := srvr.Serve(l); err != nil {
					log.Error("Stopped serving traffic", zap.Error(err))
				}
				log.Info("Stopped HTTP server")
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srvr.Close()
		},
	})
}
