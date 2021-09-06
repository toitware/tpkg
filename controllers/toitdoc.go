package controllers

import (
	"context"
	"sync"

	doc "github.com/toitware/tpkg/pkg/toitdoc"
	"github.com/toitware/tpkg/pkg/tpkg"
	"go.uber.org/zap"
)

func provideToitdoc(logger *zap.Logger, manager doc.Manager) (*toitdocCtrl, Toitdoc, error) {
	res := &toitdocCtrl{
		logger:  logger,
		manager: manager,
	}
	return res, res, nil
}

type Toitdoc interface {
	Load(ctx context.Context, desc *tpkg.Desc) (doc.Doc, error)
}

type toitdocCtrl struct {
	sync.RWMutex

	logger  *zap.Logger
	manager doc.Manager
}

func (t *toitdocCtrl) Load(ctx context.Context, desc *tpkg.Desc) (doc.Doc, error) {
	return t.manager.Get(ctx, desc)
}
