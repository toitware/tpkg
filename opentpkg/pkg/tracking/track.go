// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tracking

import "context"

type Event struct {
	Name       string
	Properties map[string]string
}
type Track func(ctx context.Context, event *Event) error

func NopTrack(ctx context.Context, event *Event) error {
	return nil
}
