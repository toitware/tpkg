package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/toitware/tpkg.git/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

type HostAddress string

func provideHostAddress(cfg *config.Config) HostAddress {
	return HostAddress(fmt.Sprintf(":%d", cfg.Port))
}

func (a HostAddress) String() string {
	return string(a)
}

func provideMux() *runtime.ServeMux {
	return runtime.NewServeMux()
}

func provideRouter() *mux.Router {
	return mux.NewRouter()
}

func provideHTTPServer(router *mux.Router, grpcServer *grpc.Server) *http.Server {
	return &http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, r)
			} else {
				router.ServeHTTP(w, r)
			}
		}), &http2.Server{}),
	}
}

func bindHTTPServer(lc fx.Lifecycle, log *zap.Logger, address HostAddress, cfg *config.Config, srv *http.Server) error {
	conn, err := net.Listen("tcp", address.String())
	if err != nil {
		return err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("Started HTTP server", zap.Stringer("address", address))
			go func() {
				if err := srv.Serve(conn); err != nil && err != http.ErrServerClosed {
					log.Fatal("Stopped serving traffic", zap.Error(err))
				}
				log.Info("Stopped HTTP server")
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return nil
}
