// Copyright (C) 2023 Toitware ApS. All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package controllers

import (
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		provideRegistry,
		provideTpkgRegistry,
		provideToitdoc,
		provideManager,
	),
	fx.Invoke(
		initRegistry,
	),
)
