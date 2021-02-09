package version

import (
	"flag"
	"fmt"
	"os"
)

var (
	// BuildDate is the date when the binary was built
	BuildDate string
	// GitCommit is the commit hash when the binary was built
	GitCommit string
	// MICVersion is the version of the MIC component
	MICVersion string
	// NMIVersion is the version of the NMI component
	NMIVersion string

	// custom user agent to append for adal and arm calls
	customUserAgent = flag.String("custom-user-agent", "", "User agent to append in addition to pod identity component and versions.")
)

// GetUserAgent is used to get the user agent string which is then provided to adal
// to use as the extended user agent header.
// The format is: aad-pod-identity/<component - either NMI or MIC>/<Version of component>/<Git commit>/<Build date>
func GetUserAgent(component, version string) string {
	if *customUserAgent != "" {
		return fmt.Sprintf("aad-pod-identity/%s/%s/%s/%s %s", component, version, GitCommit, BuildDate, *customUserAgent)
	}
	return fmt.Sprintf("aad-pod-identity/%s/%s/%s/%s", component, version, GitCommit, BuildDate)
}

// PrintVersionAndExit prints the version and exits
func PrintVersionAndExit() {
	fmt.Printf("MIC Version: %s - NMI Version: %s - Commit: %s - Date: %s\n", MICVersion, NMIVersion, GitCommit, BuildDate)
	os.Exit(0)
}
