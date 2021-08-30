// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tracking

import "context"

type TrackingEvent struct {
	Category string
	Action   string
	Label    string
	Fields   map[string]string
}
type Track func(ctx context.Context, event *TrackingEvent) error
