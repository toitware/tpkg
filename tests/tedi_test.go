// Code generated by tedi; DO NOT EDIT.

package tests

import (
	"github.com/jstroem/tedi"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	t := tedi.New(m)

	// TestLabels:
	t.TestLabel("unit")
	t.TestLabel("integration")
	t.TestLabel("regression")

	// Fixtures:
	t.Fixture(fix_Context)
	t.Fixture(fixtureCreateTestDirectory)
	t.Fixture(fixtureCreatePkgTest)

	// Tests:
	t.Test("test_toitPkg", test_toitPkg, "unit")

	os.Exit(t.Run())
}