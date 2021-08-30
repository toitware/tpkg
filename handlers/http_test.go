package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/gavv/httpexpect"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/jstroem/tedi"
	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/controllers"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func fix_gomockController(t *tedi.T) *gomock.Controller {
	ctrl := gomock.NewController(t)
	t.AfterTest(ctrl.Finish)
	return ctrl
}

func fix_ToitdocCtrl(ctrl *gomock.Controller) *controllers.MockToitdoc {
	return controllers.NewMockToitdoc(ctrl)
}

func fix_RegistryCtrl(ctrl *gomock.Controller) *controllers.MockRegistry {
	return controllers.NewMockRegistry(ctrl)
}

func fix_Logger() *zap.Logger {
	return zap.NewNop()
}

func fix_Context(t *tedi.T) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	t.AfterTest(cancel)
	return ctx
}

func fix_Config() *config.Config {
	return &config.Config{}
}

func fix_HTTPHandlers(logger *zap.Logger, cfg *config.Config, registry *controllers.MockRegistry, toitdoc *controllers.MockToitdoc) *httpHandlers {
	return provideHTTPHandlers(logger, cfg, registry, toitdoc)
}

func fix_HTTPServer(t *tedi.T, cfg *config.Config, logger *zap.Logger, handlers *httpHandlers) *httptest.Server {
	router := mux.NewRouter()
	bindHTTPHandlers(router, cfg, logger, handlers, nil)
	server := httptest.NewServer(router)
	t.AfterTest(server.Close)
	return server
}

type httpHandlerTestInput struct {
	fx.In

	Handlers *httpHandlers
	Registry *controllers.MockRegistry
	Toitdoc  *controllers.MockToitdoc
	Ctx      context.Context
	Server   *httptest.Server
}

func test_HTTPHandlers_Toitdoc(t *tedi.T) {
	t.Run("returns error if package is not found", func(t *tedi.T, i httpHandlerTestInput) {
		i.Registry.EXPECT().Package(gomock.Any(), "foo/bar/baz").Return(nil, status.Errorf(codes.NotFound, "not found"))

		e := httpexpect.New(t, i.Server.URL)
		e.GET("/foo/bar/baz/docs/").Expect().Status(http.StatusNotFound)
	})

	t.Run("returns error if version is not found", func(t *tedi.T, i httpHandlerTestInput) {
		pkg := &controllers.Package{}
		i.Registry.EXPECT().Package(gomock.Any(), "foo/bar/baz").Return(pkg, nil)

		e := httpexpect.New(t, i.Server.URL)
		e.GET("/foo/bar/baz@foo/docs/").Expect().Status(http.StatusNotFound)
	})

	t.Run("calls load on toitdoc", func(t *tedi.T, i httpHandlerTestInput) {
		desc := &tpkg.Desc{
			URL:     "foo/bar/baz",
			Version: "v1.2.3",
		}
		pkg := &controllers.Package{
			Lookup: map[string]*tpkg.Desc{
				desc.Version: desc,
			},
			Descriptions: []*tpkg.Desc{desc},
		}
		i.Registry.EXPECT().Package(gomock.Any(), "foo/bar/baz").Return(pkg, nil)
		i.Toitdoc.EXPECT().Load(gomock.Any(), desc).Return(nil, status.Errorf(codes.Unimplemented, "unimplemented"))

		e := httpexpect.New(t, i.Server.URL)
		e.GET("/foo/bar/baz/docs/").Expect().Status(http.StatusInternalServerError)
	})
}
