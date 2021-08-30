package handlers

import (
	"context"
	"fmt"

	"github.com/toitware/toit.git/tools/tpkg/build/proto/registry"
	"github.com/toitware/toit.git/tools/tpkg/config"
	"github.com/toitware/toit.git/tools/tpkg/controllers"
	"github.com/toitware/toit.git/tools/tpkg/pkg/tpkg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type registryService struct {
	registry.UnimplementedRegistryServiceServer
	logger   *zap.Logger
	registry controllers.Registry
}

var _ registry.RegistryServiceServer = (*registryService)(nil)

func provideRegistryService(logger *zap.Logger, registry controllers.Registry) *registryService {
	return &registryService{
		logger:   logger,
		registry: registry,
	}
}

func bindRegistryService(service *registryService, s *grpc.Server) {
	registry.RegisterRegistryServiceServer(s, service)
}

func (s *registryService) ListPackages(req *registry.ListPackagesRequest, stream registry.RegistryService_ListPackagesServer) error {
	packages, err := s.registry.Packages(stream.Context())
	if err != nil {
		return err
	}
	for _, p := range packages {
		d := p.Latest()
		stream.Send(&registry.ListPackagesResponse{
			Package: &registry.Package{
				Name:          d.Name,
				Url:           d.URL,
				License:       d.License,
				Description:   d.Description,
				LatestVersion: d.Version,
			},
		})
	}
	return nil
}

func (s *registryService) Sync(ctx context.Context, req *registry.SyncRequest) (*registry.SyncResponse, error) {
	if err := s.registry.Sync(ctx); err != nil {
		return nil, err
	}
	return &registry.SyncResponse{}, nil
}

func (s *registryService) GetPackageVersions(req *registry.GetPackageVersionsRequest, stream registry.RegistryService_GetPackageVersionsServer) error {
	versions, err := s.registry.Package(stream.Context(), req.Url)
	if err != nil {
		return err
	}
	for _, v := range versions.Descriptions {
		dependencies := make([]*registry.Dependency, len(v.Deps))
		for i, d := range v.Deps {
			dependencies[i] = &registry.Dependency{
				Url:     d.URL,
				Version: d.Version,
			}
		}

		stream.Send(&registry.GetPackageVersionsResponse{
			Version: &registry.PackageVersion{
				Name:         v.Name,
				Version:      v.Version,
				Description:  v.Description,
				Url:          v.URL,
				License:      v.License,
				Dependencies: dependencies,
			},
		})
	}
	return nil
}

func (s *registryService) Register(ctx context.Context, req *registry.RegisterRequest) (*registry.RegisterResponse, error) {
	url := req.Url
	version := req.Version
	if version == "" {
		return nil, status.Errorf(codes.Unimplemented, "Unimplemented for multiple versions")
	}

	if err := s.registry.RegisterPackage(ctx, url, version); err != nil {
		return nil, err
	}

	return &registry.RegisterResponse{}, nil
}

func provideCache(config *config.Config, ui tpkg.UI) tpkg.Cache {
	return tpkg.NewCache(nil, []string{config.Registry.CachePath}, ui)
}

type loggerUI struct {
	logger *zap.Logger
}

func provideLoggerUI(logger *zap.Logger) tpkg.UI {
	return &loggerUI{
		logger: logger,
	}
}

func (ui *loggerUI) ReportError(format string, a ...interface{}) error {
	ui.logger.Sugar().Errorf(format, a...)
	return fmt.Errorf(format, a...)
}

func (ui *loggerUI) ReportWarning(format string, a ...interface{}) {
	ui.logger.Sugar().Warnf(format, a...)
}

func (ui loggerUI) ReportInfo(format string, a ...interface{}) {
	ui.logger.Sugar().Infof(format, a...)
}
