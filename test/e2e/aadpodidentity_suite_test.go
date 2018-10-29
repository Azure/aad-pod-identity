package aadpodidentity

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAADPodIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	// Tell ginkgo to start executing tests defined in aadpodidentity_test.go
	RunSpecs(t, "AAD Pod Identity Suite")
}
