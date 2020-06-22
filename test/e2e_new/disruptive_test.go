// +build e2e_new

package e2e_new

import (
	. "github.com/onsi/ginkgo"
)

var _ = Describe("When AAD Pod Identity operations are disrupted", func() {
	It("should establish a new AzureAssignedIdentity and remove the old one when re-scheduling identity validator", func() {

	})

	It("should pass multiple identity validating test even when MIC is failing over", func() {

	})
})
