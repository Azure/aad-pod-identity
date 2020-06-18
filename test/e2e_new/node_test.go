// +build e2e_new

package e2e_new

import (
	. "github.com/onsi/ginkgo"
)

var _ = Describe("When there are identities assigned to the underlying nodes that are not managed by AAD Pod Identity", func() {
	It("should not delete an in use identity from a vmss", func() {

	})

	It("should delete assigned identity when identity no longer exists on underlying node", func() {

	})

	It("should not alter the system assigned identity after creating and deleting pod identity", func() {

	})

	It("should not alter the user assigned identity on VM after AAD pod identity is created and deleted", func() {

	})
})
