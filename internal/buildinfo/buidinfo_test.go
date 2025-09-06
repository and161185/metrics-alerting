// internal/buildinfo/buildinfo_test.go
package buildinfo

import "testing"

func TestPrintBuildInfo_DefaultsAndSet(t *testing.T) {
	ov, od, oc := BuildVersion, BuildDate, BuildCommit
	t.Cleanup(func() { BuildVersion, BuildDate, BuildCommit = ov, od, oc })

	BuildVersion, BuildDate, BuildCommit = "", "", ""
	PrintBuildInfo() // ветки "N/A"

	BuildVersion, BuildDate, BuildCommit = "v1", "2025-09-06", "deadbeef"
	PrintBuildInfo() // ветки "set"
}
