// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package network

import (
	"context"
	"math"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/uber-go/tally"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func provideGRPCServer(logger *zap.Logger, scope tally.Scope) *grpc.Server {
	i := newInterceptor(logger, scope)
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(math.MaxInt32),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc.StreamServerInterceptor(i.streamServerInterceptor),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc.UnaryServerInterceptor(i.unaryServerInterceptor),
		)),
	)

	return s
}

func newInterceptor(logger *zap.Logger, scope tally.Scope) *interceptor {
	return &interceptor{
		scope:  scope.SubScope("grpc"),
		logger: logger,
	}
}

type interceptor struct {
	scope  tally.Scope
	logger *zap.Logger
}

func (i *interceptor) unaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	i.logger.Debug("incoming request", zap.String("method", info.FullMethod))

	scope := i.scope.Tagged(map[string]string{
		"method":    info.FullMethod,
		"grpc_type": "unary",
	})

	scope.Counter("inbound_request").Inc(1)
	before := time.Now()

	res, err := handler(ctx, req)

	scope.Histogram("inbound_request_duration", tally.DefaultBuckets).RecordDuration(time.Since(before))
	if err == nil {
		scope.Counter("inbound_request_success").Inc(1)
		i.logger.Debug("incoming request done", zap.String("method", info.FullMethod))
	} else {
		scope.Tagged(map[string]string{"error": ErrorCode(err).String()}).Counter("inbound_request_error").Inc(1)
		i.logger.Info("incoming request error", zap.String("method", info.FullMethod), zap.Error(err))
	}

	return res, err
}

func (i *interceptor) streamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	i.logger.Debug("incoming stream request", zap.String("method", info.FullMethod))

	scope := i.scope.Tagged(map[string]string{
		"method":    info.FullMethod,
		"grpc_type": "stream",
	})

	scope.Counter("inbound_request").Inc(1)
	before := time.Now()

	err := handler(srv, ss)

	scope.Histogram("inbound_request_duration", tally.DefaultBuckets).RecordDuration(time.Since(before))
	if err == nil {
		scope.Counter("inbound_request_success").Inc(1)
	} else {
		scope.Tagged(map[string]string{"error": ErrorCode(err).String()}).Counter("inbound_request_error").Inc(1)
		i.logger.Error("incoming stream request error", zap.Error(err), zap.String("method", info.FullMethod))
	}

	return err
}
