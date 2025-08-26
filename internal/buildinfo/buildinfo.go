package buildinfo

import "fmt"

func PrintBuildInfo(buildVersion, buildDate, buildCommit string) {
	v := buildVersion
	if v == "" {
		v = "N/A"
	}
	d := buildDate
	if d == "" {
		d = "N/A"
	}
	c := buildCommit
	if c == "" {
		c = "N/A"
	}

	fmt.Printf("Build version: %s\n", v)
	fmt.Printf("Build date: %s\n", d)
	fmt.Printf("Build commit: %s\n", c)
}
