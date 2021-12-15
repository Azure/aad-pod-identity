//go:build e2e
// +build e2e

package helm

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Azure/aad-pod-identity/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/mod/semver"
)

const (
	chartName = "aad-pod-identity"
)

// InstallInput is the input for Install.
type InstallInput struct {
	Config         *framework.Config
	NamespacedMode bool
}

// Install installs aad-pod-identity via Helm 3.
func Install(input InstallInput) {
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Helm.Install")

	cwd, err := os.Getwd()
	Expect(err).To(BeNil())

	// Change current working directory to repo root
	// Before installing aad-pod identity through Helm
	os.Chdir("../..")
	defer os.Chdir(cwd)

	args := append([]string{
		"install",
		chartName,
		"manifest_staging/charts/aad-pod-identity",
		"--wait",
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	})
	args = append(args, generateValueArgs(input.Config)...)

	err = helm(args)
	Expect(err).To(BeNil())
}

// Uninstall uninstalls aad-pod-identity via Helm 3.
func Uninstall() {
	args := []string{
		"uninstall",
		chartName,
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	}

	// ignore error to allow cleanup completion
	_ = helm(args)
}

// UpgradeInput is the input for Upgrade.
type UpgradeInput struct {
	Config *framework.Config
}

// Upgrade upgrades aad-pod-identity via Helm 3.
func Upgrade(input UpgradeInput) {
	Expect(input.Config).NotTo(BeNil(), "input.Config is required for Helm.Upgrade")

	cwd, err := os.Getwd()
	Expect(err).To(BeNil())

	// Change current working directory to repo root
	// Before installing aad-pod identity through Helm
	os.Chdir("../..")
	defer os.Chdir(cwd)

	args := append([]string{
		"upgrade",
		chartName,
		"manifest_staging/charts/aad-pod-identity",
		"--reuse-values",
		"--wait",
		fmt.Sprintf("--namespace=%s", framework.NamespaceKubeSystem),
		"--debug",
	})
	args = append(args, generateValueArgs(input.Config)...)

	err = helm(args)
	Expect(err).To(BeNil())
}

func generateValueArgs(config *framework.Config) []string {
	args := []string{
		fmt.Sprintf("--set=image.repository=%s", config.Registry),
		fmt.Sprintf("--set=mic.tag=%s", config.MICVersion),
		fmt.Sprintf("--set=nmi.tag=%s", config.NMIVersion),
		fmt.Sprintf("--set=mic.syncRetryDuration=%s", config.MICSyncInterval),
		fmt.Sprintf("--set=nmi.retryAttemptsForCreated=%d", config.RetryAttemptsForCreated),
		fmt.Sprintf("--set=nmi.retryAttemptsForAssigned=%d", config.RetryAttemptsForAssigned),
		fmt.Sprintf("--set=nmi.findIdentityRetryIntervalInSeconds=%d", config.FindIdentityRetryIntervalInSeconds),
		// Setting this explicitly as the old charts don't have this value object and fail with --reuse-values
		fmt.Sprintf("--set=mic.customCloud.enabled=false"),
	}

	// TODO (aramase) bump this to compare against v1.7.3 after next release
	if semver.Compare(config.MICVersion, "v1.7.2") > 1 && semver.Compare(config.NMIVersion, "v1.7.2") > 1 {
		args = append(args, fmt.Sprintf("--set=customUserAgent=pi-e2e", "--set=mic.logVerbosity=9"))
	}

	if config.ImmutableUserMSIs != "" {
		args = append(args, fmt.Sprintf("--set=mic.immutableUserMSIs=%s", config.ImmutableUserMSIs))
	}

	if config.NMIMode == "managed" {
		args = append(args, fmt.Sprintf("--set=operationMode=%s", "managed"))
	}

	if config.BlockInstanceMetadata {
		args = append(args, fmt.Sprintf("--set=nmi.blockInstanceMetadata=%t", config.BlockInstanceMetadata))
	}

	if config.MetadataHeaderRequired {
		args = append(args, fmt.Sprintf("--set=nmi.metadataHeaderRequired=%t", config.MetadataHeaderRequired))
	}

	if config.IdentityReconcileInterval != 0 {
		args = append(args, fmt.Sprintf("--set=mic.identityAssignmentReconcileInterval=%s", config.IdentityReconcileInterval))
	}

	if config.SetRetryAfterHeader {
		args = append(args, fmt.Sprintf("--set=nmi.setRetryAfterHeader=%t", config.SetRetryAfterHeader))
	}

	return args
}

func helm(args []string) error {
	By(fmt.Sprintf("helm %s", strings.Join(args, " ")))

	cmd := exec.Command("helm", args...)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s", stdoutStderr)

	return err
}
