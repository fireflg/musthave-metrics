package buildinfo_test

import (
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/buildinfo"
)

func ExamplePrint() {
	buildinfo.Print()
	// Output:
	// Build version: N/A
	// Build date: N/A
	// Build commit: N/A
}

func TestPrintWithLdflagsValues(t *testing.T) {
	oldVersion, oldDate, oldCommit := buildinfo.Version, buildinfo.Date, buildinfo.Commit
	defer func() {
		buildinfo.Version, buildinfo.Date, buildinfo.Commit = oldVersion, oldDate, oldCommit
	}()

	buildinfo.Version, buildinfo.Date, buildinfo.Commit = "v1.0.0", "2026-01-01", "abc123"
	buildinfo.Print()
}
