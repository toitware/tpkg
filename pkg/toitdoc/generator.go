package toitdoc

import (
	"context"
	"os/exec"

	"github.com/toitlang/tpkg/pkg/tpkg"
	"github.com/toitware/tpkg/config"
	"go.uber.org/zap"
)

type generator struct {
	logger *zap.Logger
	cfg    config.SDK
}

func provideGenerator(cfg *config.Config, logger *zap.Logger) *generator {
	return &generator{
		logger: logger,
		cfg:    cfg.Toitdocs.SDK,
	}
}

func (g *generator) generateDocs(ctx context.Context, projectPath string, desc *tpkg.Desc, outFile string) error {
	cmd := exec.CommandContext(ctx, g.cfg.ToitlspPath(),
		"toitdoc",
		"--toitc", g.cfg.ToitcPath(),
		"--sdk", g.cfg.Path,
		"--exclude-sdk",
		"--out", outFile,
		"--version", desc.Version,
		"./src",
	)
	cmd.Dir = projectPath

	if err := cmd.Run(); err != nil {
		var stdout string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdout = string(exitErr.Stderr)
		}
		g.logger.Error("failed to generate toitdocs", zap.String("stdout", stdout), zap.String("cwd", projectPath), zap.String("cmd", cmd.String()), zap.Error(err), zap.String("url", desc.URL), zap.String("version", desc.Version))
		return err
	}

	return nil
}
