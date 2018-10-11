package aadpodidentity

import (
	"github.com/Azure/aad-pod-identity/test/e2e/azureassignedidentity"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	It("should create an AzureAssignedIdentity", func() {
		azureAssignedIdentity, err := azureassignedidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(azureAssignedIdentity.AzureAssignedIdentities)).To(Equal(1))
	})
})
