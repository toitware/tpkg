// Copyright (C) 2021 Toitware ApS. All rights reserved.

package commands

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func FirstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func ErrorMessage(err error) string {
	return status.Convert(err).Message()
}

func IsAlreadyExistsError(err error) bool {
	return status.Code(err) == codes.AlreadyExists
}
