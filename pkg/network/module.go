// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package network

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		provideGRPCServer,
		provideMux,
		provideHTTPServer,
		provideRouter,
		provideHostAddress,
	),
	fx.Invoke(
		bindHTTPServer,
	),
)
