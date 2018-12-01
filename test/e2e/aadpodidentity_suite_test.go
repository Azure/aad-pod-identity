package aadpodidentity

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAADPodIdentity(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AAD Pod Identity Suite")
}
