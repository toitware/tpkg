// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package controllers

import (
	"context"
	"sync"

	"github.com/toitlang/tpkg/pkg/tpkg"
	doc "github.com/toitware/tpkg/pkg/toitdoc"
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
