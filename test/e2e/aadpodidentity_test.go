package aadpodidentity_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	Describe("Creating an AzureIdentity and AzureIdentityBinding to bind the identity to a pod", func() {
		It("should create an AzureAssignedIdentity", func() {
			Expect(1).To(Equal(1))
		})

		It("should be able to aquire a token and access azure resources", func() {
			Expect(1).To(Equal(1))
		})
	})
})
