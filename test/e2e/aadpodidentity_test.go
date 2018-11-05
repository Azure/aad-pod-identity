package aadpodidentity

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/azure"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/common/k8s/deploy"
	"github.com/Azure/aad-pod-identity/test/common/k8s/node"
	"github.com/Azure/aad-pod-identity/test/common/k8s/pod"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/Azure/aad-pod-identity/test/e2e/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
)

var _ = BeforeSuite(func() {
	fmt.Println("Setting up the test suite environment...")

	// Create a folder '_output' in template/ for storing temporary deployment files
	err := os.Mkdir(templateOutputPath, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	c, err := config.ParseConfig()
	Expect(err).NotTo(HaveOccurred())
	cfg = *c // To avoid 'Declared and not used' linting error

	// Install CRDs and deploy MIC and NMI
	cmd := exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	fmt.Println("\nTearing down the test suite environment...")

	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("kubectl", "delete", "deploy", "--all")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	BeforeEach(func() {
		fmt.Println("Setting up the test environment...")

		// Uncordon every node in case of failed test #5
		node.UncordonAll()

		// TODO: Start all VMs in case of failed test #7
	})

	AfterEach(func() {
		fmt.Println("\nTearing down the test environment...")

		cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
		util.PrintCommand(cmd)
		_, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	It("should pass the identity validating test", func() {
		setUpIdentityAndDeployment("")

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())

		validateAzureAssignedIdentity(azureAssignedIdentity, "test-identity")
	})

	It("should not pass the identity validating test if the AzureIdentity is deleted", func() {
		setUpIdentityAndDeployment("")

		err := azureidentity.DeleteOnCluster("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		identityClientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		Expect(identityClientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", identityClientID, "--resourcegroup", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).To(HaveOccurred())
	})

	It("should not pass the identity validating test if the AzureIdentityBinding is deleted", func() {
		setUpIdentityAndDeployment("")

		err := azureidentitybinding.Delete("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		identityClientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		Expect(identityClientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", identityClientID, "--resourcegroup", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).To(HaveOccurred())
	})

	It("should delete the AzureAssignedIdentity if the deployment is deleted", func() {
		setUpIdentityAndDeployment("")

		waitForDeployDeletion("identity-validator")

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	It("should establish a new AzureAssignedIdentity and remove the old one when draining the node containing identity validator", func() {
		setUpIdentityAndDeployment("")

		podName, err := pod.GetNameByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		// Get the name of the node to drain
		nodeName, err := pod.GetNodeName(podName)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeName).NotTo(Equal(""))

		// Drain the node that contains identity validator
		node.Drain(nodeName)

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(ok).To(Equal(true))
		Expect(err).NotTo(HaveOccurred())

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())

		// Make sure the AzureAssignedIdentity is updated along with the new pod
		validateAzureAssignedIdentity(azureAssignedIdentity, "test-identity")

		node.Uncordon(nodeName)

		waitForDeployDeletion("identity-validator")
	})

	It("should remove the correct identities when adding AzureIdentity and AzureIdentityBinding in order and removing them in random order", func() {
		testData := make([]struct {
			identityName          string
			identityClientID      string
			identityValidatorName string
			azureAssignedIdentity aadpodid.AzureAssignedIdentity
		}, 5)

		for i := 0; i < 5; i++ {
			identityName := fmt.Sprintf("test-identity-%d", i)
			identityValidatorName := fmt.Sprintf("identity-validator-%d", i)

			setUpIdentityAndDeployment(fmt.Sprintf("%d", i))

			identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, identityName)
			Expect(err).NotTo(HaveOccurred())
			Expect(identityClientID).NotTo(Equal(""))

			azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidatorName)
			Expect(err).NotTo(HaveOccurred())

			validateAzureAssignedIdentity(azureAssignedIdentity, identityName)

			testData[i] = struct {
				identityName          string
				identityClientID      string
				identityValidatorName string
				azureAssignedIdentity aadpodid.AzureAssignedIdentity
			}{
				identityName,
				identityClientID,
				identityValidatorName,
				azureAssignedIdentity,
			}
		}

		// Shuffle the test data
		for i := range testData {
			j := rand.Intn(i + 1)
			testData[i], testData[j] = testData[j], testData[i]
		}

		// Delete i-th elements in test data and check if the identities beyond index i are still functioning
		for i, data := range testData {
			azureidentity.DeleteOnCluster(data.identityName, templateOutputPath)
			azureassignedidentity.WaitOnLengthMatched(5 - 1 - i)

			// Make sure that the identity validator cannot access to the resource group anymore
			cmd := exec.Command("kubectl", "exec", data.azureAssignedIdentity.Spec.Pod, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", data.identityClientID, "--resourcegroup", cfg.ResourceGroup)
			_, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred())
			waitForDeployDeletion(data.identityValidatorName)

			// Make sure that the existing identities are still functioning
			for j := i + 1; j < 5; j++ {
				validateAzureAssignedIdentity(testData[j].azureAssignedIdentity, testData[j].identityName)
			}
		}
	})

	// It("should re-schedule the identity validator and its identity to a new node after powering down and restarting the node containing them", func() {
	// 	setUpIdentityAndDeployment("")

	// 	identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, "test-identity")
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(identityClientID).NotTo(Equal(""))

	// 	podName, err := pod.GetNameByPrefix("identity-validator")
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(podName).NotTo(Equal(""))

	// 	// Get the name of the node to drain
	// 	nodeName, err := pod.GetNodeName(podName)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(nodeName).NotTo(Equal(""))

	// 	azure.StopVM(cfg.ResourceGroup, nodeName)

	// 	// 2 AzureAssignedIdentity, one for the powered down node, one for the new node
	// 	ok, err := azureassignedidentity.WaitOnLengthMatched(2)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(ok).To(Equal(true))

	// 	list, err := azureassignedidentity.GetAll()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Get the new pod name
	// 	podName, err = pod.GetNameByPrefix("identity-validator")
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(podName).NotTo(Equal(""))

	// 	// TODO: Get the new azure assigned identity
	// 	// azureassignedidentity.GetByPrefix(podName)
	// 	// Make sure the AzureAssignedIdentity is updated along with the new pod after powering down its old node
	// 	validateAzureAssignedIdentity(list.Items[0], podName, "test-identity", identityClientID)

	// 	// Start the VM again to ensure that the old AzureAssignedIdentity is deleted
	// 	azure.StartVM(cfg.ResourceGroup, nodeName)

	// 	ok, err = azureassignedidentity.WaitOnLengthMatched(1)
	// 	Expect(ok).To(Equal(true))
	// 	Expect(err).NotTo(HaveOccurred())

	// 	list, err = azureassignedidentity.GetAll()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Final validation to ensure that everything is functioning after restarting the node
	// 	validateAzureAssignedIdentity(list.Items[0], podName, "test-identity", identityClientID)
	// })
})

// setUpIdentityAndDeployment will deploy AzureIdentity, AzureIdentityBinding, and an identity validator
// Suffix will give the tests the option to add a suffix to the end of the identity name, useful for scale tests
func setUpIdentityAndDeployment(suffix string) {
	azureIdentityName := "test-identity"
	identityValidatorName := "identity-validator"

	if suffix != "" {
		azureIdentityName += "-" + suffix
		identityValidatorName += "-" + suffix
	}

	err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	err = azureidentitybinding.Create(azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	err = deploy.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, identityValidatorName, azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	ok, err := deploy.WaitOnReady(identityValidatorName)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
}

// validateAzureAssignedIdentity will make sure a given AzureAssignedIdentity has the correct properties
func validateAzureAssignedIdentity(azureAssignedIdentity aadpodid.AzureAssignedIdentity, identityName string) {
	identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, identityName)
	Expect(err).NotTo(HaveOccurred())
	Expect(identityClientID).NotTo(Equal(""))

	podName := azureAssignedIdentity.Spec.Pod
	Expect(podName).NotTo(Equal(""))

	// The Azure Assigned Identity name should be "<pod name>-<namespace>-<identity name>"
	Expect(azureAssignedIdentity.ObjectMeta.Name).To(Equal(fmt.Sprintf("%s-%s-%s", podName, "default", identityName)))

	// Assert Azure Identity Binding properties
	Expect(azureAssignedIdentity.Spec.AzureBindingRef.ObjectMeta.Name).To(Equal(fmt.Sprintf("%s-binding", identityName)))
	Expect(azureAssignedIdentity.Spec.AzureBindingRef.ObjectMeta.Namespace).To(Equal("default"))
	Expect(azureAssignedIdentity.Spec.AzureBindingRef.Spec.AzureIdentity).To(Equal(identityName))
	Expect(azureAssignedIdentity.Spec.AzureBindingRef.Spec.Selector).To(Equal(identityName))

	// Assert Azure Identity properties
	Expect(azureAssignedIdentity.Spec.AzureIdentityRef.ObjectMeta.Name).To(Equal(identityName))
	Expect(azureAssignedIdentity.Spec.AzureIdentityRef.ObjectMeta.Namespace).To(Equal("default"))
	Expect(azureAssignedIdentity.Spec.AzureIdentityRef.Spec.ClientID).To(Equal(identityClientID))

	cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscriptionid", cfg.SubscriptionID, "--clientid", identityClientID, "--resourcegroup", cfg.ResourceGroup)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

// waitForDeployDeletion will block until a give deploy and its pods are completed deleted
func waitForDeployDeletion(deployName string) {
	err := deploy.Delete(deployName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	ok, err := pod.WaitOnDeletion(deployName)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
}
