// Copyright (C) 2021 Toitware ApS. All rights reserved.

package tpkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Solver(t *testing.T) {
	t.Run("Solve Transitive", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{

			{
				URL:     "b",
				Version: "^1.0.0",
			},
		})
		b11 := NewDesc("b", "", "b", "1.1.0", "MIT", "", []descPackage{

			{
				URL:     "c",
				Version: ">=2.0.0,<3.1.2",
			},
		})
		c2 := NewDesc("c", "", "c", "2.0.5", "MIT", "", []descPackage{})


		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b11, c2},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		solution, err := solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		require.NoError(t, err)
		assert.Len(t, solution, 3)
		aVersions, ok := solution["a"]
		assert.True(t, ok)
		assert.Len(t, aVersions, 1)
		assert.Contains(t, aVersions, a1.Version)
		bVersions, ok := solution["b"]
		assert.True(t, ok)
		assert.Len(t, bVersions, 1)
		assert.Contains(t, bVersions, b11.Version)
		cVersions, ok := solution["c"]
		assert.True(t, ok)
		assert.Len(t, cVersions, 1)
		assert.Contains(t, cVersions, c2.Version)
	})

	t.Run("Solve Correct Version", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{

			{
				URL:     "b",
				Version: "^1.0.0",
			},
		})
		b01 := NewDesc("b", "", "b", "0.1.0", "MIT", "", []descPackage{})
		b11 := NewDesc("b", "", "b", "1.1.0", "MIT", "", []descPackage{})
		b21 := NewDesc("b", "", "b", "2.1.0", "MIT", "", []descPackage{})


		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b01, b11, b21},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		solution, err := solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		require.NoError(t, err)
		assert.Len(t, solution, 2)
		aVersions, ok := solution["a"]
		assert.True(t, ok)
		assert.Len(t, aVersions, 1)
		assert.Contains(t, aVersions, a1.Version)
		bVersions, ok := solution["b"]
		assert.True(t, ok)
		assert.Len(t, bVersions, 1)
		assert.Contains(t, bVersions, b11.Version)
	})

	t.Run("Solve Highest Version", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{

			{
				URL:     "b",
				Version: "^1.0.0",
			},
		})
		b111 := NewDesc("b", "", "b", "1.1.1", "MIT", "", []descPackage{})
		b123 := NewDesc("b", "", "b", "1.2.3", "MIT", "", []descPackage{})
		b21 := NewDesc("b", "", "b", "2.1.0", "MIT", "", []descPackage{})


		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b111, b123, b21},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		solution, err := solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		require.NoError(t, err)
		assert.Len(t, solution, 2)
		aVersions, ok := solution["a"]
		assert.True(t, ok)
		assert.Len(t, aVersions, 1)
		assert.Contains(t, aVersions, a1.Version)
		bVersions, ok := solution["b"]
		assert.True(t, ok)
		assert.Len(t, bVersions, 1)
		assert.Contains(t, bVersions, b123.Version)
	})

	t.Run("Solve Multiple Version", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{

			{
				URL:     "b",
				Version: "^1.0.0",
			},
			{
				URL:     "c",
				Version: "^1.0.0",
			},
		},
		)
		b111 := NewDesc("b", "", "b", "1.1.1", "MIT", "", []descPackage{

			{
				URL:     "c",
				Version: "^2.0.0",
			},
		},
		)
		c1 := NewDesc("c", "", "c", "1.2.3", "MIT", "", []descPackage{})
		c2 := NewDesc("c", "", "c", "2.3.4", "MIT", "", []descPackage{})


		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b111, c1, c2},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		solution, err := solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		require.NoError(t, err)
		assert.Len(t, solution, 3)
		aVersions, ok := solution["a"]
		assert.True(t, ok)
		assert.Len(t, aVersions, 1)
		assert.Contains(t, aVersions, a1.Version)
		bVersions, ok := solution["b"]
		assert.True(t, ok)
		assert.Len(t, bVersions, 1)
		assert.Contains(t, bVersions, b111.Version)
		cVersions, ok := solution["c"]
		assert.True(t, ok)
		assert.Len(t, cVersions, 2)
		assert.Contains(t, cVersions, c1.Version)
		assert.Contains(t, cVersions, c2.Version)
	})

	t.Run("Solve Cycle", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{


			{
				URL:     "b",
				Version: "^1.0.0",
			},
		},
		)
		b111 := NewDesc("b", "", "b", "1.1.1", "MIT", "", []descPackage{


			{
				URL:     "a",
				Version: "^1.0.0",
			},
		},
		)

		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b111},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		solution, err := solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		require.NoError(t, err)
		assert.Len(t, solution, 2)
		aVersions, ok := solution["a"]
		assert.True(t, ok)
		assert.Len(t, aVersions, 1)
		assert.Contains(t, aVersions, a1.Version)
		bVersions, ok := solution["b"]
		assert.True(t, ok)
		assert.Len(t, bVersions, 1)
		assert.Contains(t, bVersions, b111.Version)
		assert.True(t, ok)
	})

	t.Run("Fail Missing Pkg", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{


			{
				URL:     "b",
				Version: "^1.0.0",
			},
		},
		)

		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		_, err = solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		assert.True(t, IsErrAlreadyReported(err))
		assert.Len(t, ui.messages, 1)
		assert.Equal(t, "Error: Package 'b' not found", ui.messages[0])
	})

	t.Run("Fail Version", func(t *testing.T) {
		ui := testUI{}
		a1 := NewDesc("a", "", "a", "1.7.0", "MIT", "", []descPackage{


			{
				URL:     "b",
				Version: "^1.0.0",
			},
		},
		)

		b234 := NewDesc("b", "", "b", "2.3.4", "MIT", "", []descPackage{})


		registry := pathRegistry{
			path:    "not important",
			entries: []*Desc{a1, b234},
		}
		registries := Registries{
			&registry,
		}
		solver, err := NewSolver(registries, &ui)
		require.NoError(t, err)
		startConstraint, err := parseConstraint(a1.Version)
		require.NoError(t, err)
		_, err = solver.Solve([]SolverDep{
			{
				url:         "a",
				constraints: startConstraint,
			},
		})
		assert.True(t, IsErrAlreadyReported(err))
		assert.Len(t, ui.messages, 1)
		assert.Equal(t, "Error: No version of 'b' satisfying '>=1.0.0,<2.0.0'", ui.messages[0])
	})
}