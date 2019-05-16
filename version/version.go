package version

import (
	"fmt"
	"os"
)

// BuildDate is the date when the binary was built
var BuildDate string

// GitCommit is the commit hash when the binary was built
var GitCommit string

// MICVersion is the version of the MIC component
var MICVersion string

// NMIVersion is the version of the NMI component
var NMIVersion string

// PrintVersionAndExit prints the version and exits
func PrintVersionAndExit() {
	fmt.Printf("MIC Version: %s - NMI Version: %s - Commit: %s - Date: %s\n", MICVersion, NMIVersion, GitCommit, BuildDate)
	os.Exit(0)
}
