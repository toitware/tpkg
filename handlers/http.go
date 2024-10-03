// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/toitware/tpkg/build/proto/registry"
	"github.com/toitware/tpkg/config"
	"github.com/toitware/tpkg/controllers"
	"github.com/toitware/tpkg/pkg/network"
	doc "github.com/toitware/tpkg/pkg/toitdoc"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func proxyHTTPToGRPC(lc fx.Lifecycle, router *mux.Router, mux *runtime.ServeMux, address network.HostAddress, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())

	var options []grpc.DialOption
	options = append(options, grpc.WithInsecure())

	registry.RegisterRegistryServiceHandlerFromEndpoint(ctx, mux, address.String(), options)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			return nil
		},
	})

}

type httpHandlers struct {
	logger      *zap.Logger
	registry    controllers.Registry
	toitdoc     controllers.Toitdoc
	toitdocCfg  config.Toitdocs
	webFilePath string
}

func provideHTTPHandlers(logger *zap.Logger, cfg *config.Config, registry controllers.Registry, toitdoc controllers.Toitdoc) *httpHandlers {
	return &httpHandlers{
		logger:      logger,
		registry:    registry,
		toitdoc:     toitdoc,
		toitdocCfg:  cfg.Toitdocs,
		webFilePath: cfg.WebPath,
	}
}

func bindHTTPHandlers(router *mux.Router, cfg *config.Config, logger *zap.Logger, h *httpHandlers, apiHandler *runtime.ServeMux) {
	router.NotFoundHandler = network.HTTPHandle(h.web)
	router.Handle("/{package:[^@]+}/docs/{path:.*}", network.HTTPHandle(h.toitdocs))
	router.Handle("/{package:[^@]+}@{version:[^/]+}/docs/{path:.*}", network.HTTPHandle(h.toitdocs))
	router.PathPrefix("/api/").Handler(http.StripPrefix("/api", apiHandler))
	router.Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middlewares := []mux.MiddlewareFunc{
		handlers.CORS(
			handlers.AllowedHeaders([]string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"}),
			handlers.AllowedMethods([]string{"GET", "POST", "DELETE", "HEAD", "OPTIONS"}),
			handlers.AllowedOriginValidator(func(host string) bool { return true }),
			handlers.AllowCredentials(),
		),
	}

	if cfg.HTTPS {
		redirect := redirectHTTPToHTTPS(logger)
		middlewares = append(middlewares, redirect)
		router.NotFoundHandler = redirect(router.NotFoundHandler)
	}

	router.Use(middlewares...)
}

func redirectHTTPToHTTPS(log *zap.Logger) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// http -> https
			if r.Header.Get("X-Forwarded-Proto") == "http" && r.URL.Path != "/health" {
				url := *r.URL
				url.Host = r.Host
				url.Scheme = "https"
				log.Info("http to https redirect", zap.String("url", url.String()))
				http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}

func parsePackage(identifier string) (pkg, version, path string) {
	if parts := strings.SplitN(identifier, "@", 2); len(parts) == 2 {
		pkg = parts[0]
		version = parts[1]
	} else {
		pkg = identifier
	}
	return
}

func (h *httpHandlers) toitdocs(rw http.ResponseWriter, r *http.Request) error {
	pkgName := mux.Vars(r)["package"]
	pkg, err := h.registry.Package(r.Context(), pkgName)
	if err != nil {
		return err
	}
	version, noVersion := mux.Vars(r)["version"]
	if !noVersion {
		version = pkg.Latest().Version
	}

	desc, ok := pkg.Lookup[version]
	if !ok {
		return status.Errorf(codes.NotFound, "package '%s' did not have a version '%s'", pkgName, version)
	}

	path := mux.Vars(r)["path"]
	h.logger.Debug("Serving toitdoc for package", zap.String("package", desc.URL), zap.String("version", desc.Version), zap.String("path", path))
	doc, err := h.toitdoc.Load(r.Context(), desc)
	if err != nil {
		h.logger.Error("failed to load toitdoc", zap.Error(err), zap.String("package", desc.URL), zap.String("version", desc.Version), zap.String("path", path))
		return status.Errorf(codes.Internal, "failed to load package '%s@%s", desc.URL, desc.Version)
	}

	srv := &toitdocFileServer{
		doc:        doc,
		viewerPath: h.toitdocCfg.ViewerPath,
	}
	return srv.serve(rw, r, path)
}

func (h *httpHandlers) web(rw http.ResponseWriter, r *http.Request) error {
	p := strings.Trim(r.URL.Path, "/")

	if p == "" {
		p = "index.html"
	}
	if strings.HasSuffix(p, "/") {
		p += "index.html"
	}

	p = filepath.Join(h.webFilePath, p)
	mainIndexPath := filepath.Join(h.webFilePath, "index.html")
	err := serveFile(rw, r, p, mainIndexPath)
	if status.Code(err) == codes.NotFound {
		http.ServeFile(rw, r, mainIndexPath)
		return nil
	}
	return err
}

type toitdocFileServer struct {
	doc        doc.Doc
	viewerPath string
}

func (t *toitdocFileServer) serve(rw http.ResponseWriter, r *http.Request, name string) error {
	if name == "" {
		name = "index.html"
	}
	// Never serve directories.
	if strings.HasSuffix(name, "/") {
		name += "index.html"
	}
	if name == "index.html" {
		http.ServeFile(rw, r, t.doc.ViewerIndexPath())
		return nil
	}

	if name == "toitdoc.json" {
		http.ServeFile(rw, r, t.doc.JSONPath())
		return nil
	}

	return serveFile(rw, r, filepath.Join(t.viewerPath, name), t.doc.ViewerIndexPath())
}

func serveFile(rw http.ResponseWriter, r *http.Request, path string, mainIndexPath string) error {
	stat, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if err == nil {
			// If its not a directory serve the file.
			if !stat.IsDir() {
				http.ServeFile(rw, r, path)
				return nil
			}
		} else {
			return err
		}
	}

	// If the path has no ext. try and serve the index file from the directory.
	if filepath.Ext(path) == "" {
		path := filepath.Join(path, "index.html")
		if _, err := os.Stat(path); err == nil {
			http.ServeFile(rw, r, path)
		} else {
			http.ServeFile(rw, r, mainIndexPath)
		}
		return nil
	}

	return status.Error(codes.NotFound, "404 page not found")
}
