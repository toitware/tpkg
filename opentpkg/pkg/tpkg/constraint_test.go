// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseConstraintRange(t *testing.T) {
	tests := [][]string{
		{"0", ">=0,<1.0.0"},
		{"1", ">=1,<2.0.0"},
		{"0.5", ">=0.5,<0.6.0"},
		{"1.5", ">=1.5,<1.6.0"},
		{"0.5.3", "0.5.3"},
		{"1.5.3", "1.5.3"},
		{"1.5.3-alpha", "1.5.3-alpha"},
		{"0.0.1.4.5", "0.0.1.4.5"},
	}
	for _, test := range tests {
		t.Run(test[0], func(t *testing.T) {
			in := test[0]
			expectedIn := test[1]
			actual, err := parseInstallConstraint(in)
			require.NoError(t, err)
			expected, err := version.NewConstraint(expectedIn)
			require.NoError(t, err)
			assert.Equal(t, expected.String(), actual.String())
		})
	}
}
