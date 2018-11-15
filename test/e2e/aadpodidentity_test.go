package aadpodidentity

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/Azure/aad-pod-identity/test/common/azure"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/common/k8s/deploy"
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
	fmt.Println("Tearing down the test suite environment...")

	// Uninstall CRDs and delete MIC and NMI
	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	BeforeEach(func() {
		fmt.Println("Setting up the test environment...")

		err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = azureidentitybinding.Create("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = deploy.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, "identity-validator", "test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := deploy.WaitOnReady("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	AfterEach(func() {
		fmt.Println("Tearing down the test environment...")

		cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
		util.PrintCommand(cmd)
		_, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnDeletion()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	It("should pass the identity validating test", func() {
		clientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		Expect(clientID).NotTo(Equal(""))

		list, err := azureassignedidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Items).To(HaveLen(1))
		Expect(list.Items[0].Spec.Pod).NotTo(Equal(""))
		podName := list.Items[0].Spec.Pod

		// The Azure Assigned Identity name should be "<pod name>-<namespace>-<identity name>"
		Expect(list.Items[0].ObjectMeta.Name).To(Equal(fmt.Sprintf("%s-%s-%s", podName, "default", "test-identity")))

		// Assert Azure Identity Binding properties
		Expect(list.Items[0].Spec.AzureBindingRef.ObjectMeta.Name).To(Equal("test-identity-binding"))
		Expect(list.Items[0].Spec.AzureBindingRef.ObjectMeta.Namespace).To(Equal("default"))
		Expect(list.Items[0].Spec.AzureBindingRef.Spec.AzureIdentity).To(Equal("test-identity"))
		Expect(list.Items[0].Spec.AzureBindingRef.Spec.Selector).To(Equal("test-identity"))

		// Assert Azure Identity properties
		Expect(list.Items[0].Spec.AzureIdentityRef.ObjectMeta.Name).To(Equal("test-identity"))
		Expect(list.Items[0].Spec.AzureIdentityRef.ObjectMeta.Namespace).To(Equal("default"))
		Expect(list.Items[0].Spec.AzureIdentityRef.Spec.ClientID).To(Equal(clientID))

		cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscription-id", cfg.SubscriptionID, "--identity-client-id", clientID, "--resource-group", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not pass the identity validating test if the AzureIdentity is deleted", func() {
		err := azureidentity.DeleteOnCluster("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		list, err := azureidentity.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Items).To(HaveLen(0))

		ok, err := azureassignedidentity.WaitOnDeletion()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		clientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		Expect(clientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscription-id", cfg.SubscriptionID, "--identity-client-id", clientID, "--resource-group", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).To(HaveOccurred())
	})

	It("should not pass the identity validating test if the AzureIdentityBinding is deleted", func() {
		err := azureidentitybinding.Delete("test-identity", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		list, err := azureidentitybinding.GetAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(list.Items).To(HaveLen(0))

		ok, err := azureassignedidentity.WaitOnDeletion()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		clientID, err := azureidentity.GetClientID(cfg.ResourceGroup, "test-identity")
		Expect(err).NotTo(HaveOccurred())
		Expect(clientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix("identity-validator")
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		cmd := exec.Command("kubectl", "exec", podName, "--", "identityvalidator", "--subscription-id", cfg.SubscriptionID, "--identity-client-id", clientID, "--resource-group", cfg.ResourceGroup)
		_, err = cmd.CombinedOutput()
		Expect(err).To(HaveOccurred())
	})

	It("should delete the AzureAssignedIdentity if the deployment is deleted", func() {
		err := deploy.Delete("identity-validator", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnDeletion()
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})
})
