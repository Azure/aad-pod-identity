package aadpodidentity

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/test/common/azure"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/common/k8s/daemonset"
	"github.com/Azure/aad-pod-identity/test/common/k8s/deploy"
	"github.com/Azure/aad-pod-identity/test/common/k8s/infra"
	"github.com/Azure/aad-pod-identity/test/common/k8s/node"
	"github.com/Azure/aad-pod-identity/test/common/k8s/pod"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/Azure/aad-pod-identity/test/e2e/config"
	"github.com/pkg/errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	clusterIdentity   = "cluster-identity"
	keyvaultIdentity  = "keyvault-identity"
	identityValidator = "identity-validator"
	nmiDaemonSet      = "nmi"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
)

var _ = BeforeSuite(func() {
	fmt.Println("Setting up the test suite environment...")
	fmt.Println("Creating directory ", templateOutputPath)

	// Create a folder '_output' in template/ for storing temporary deployment files
	err := os.Mkdir(templateOutputPath, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	c, err := config.ParseConfig()
	Expect(err).NotTo(HaveOccurred())
	cfg = *c

	setupInfra()
})

var _ = AfterSuite(func() {
	fmt.Println("\nTearing down the test suite environment...")

	// Uninstall CRDs and delete MIC and NMI
	err := deploy.Delete("default", templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command("kubectl", "delete", "-f", "template/busyboxds.yaml")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	cmd = exec.Command("kubectl", "delete", "deploy", "--all")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	err = os.RemoveAll(templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	nodeList, err := node.GetAll()
	Expect(err).NotTo(HaveOccurred())

	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}

		// TODO: Start all VMs in case of failed test #7
		// Uncordon every node in case of failed test #5
		err := node.Uncordon(n.Name)
		Expect(err).NotTo(HaveOccurred())

		if strings.Contains(n.Name, "master") {
			continue
		}

		// Delete all user assigned identities
		userAssignedIdentities, err := azure.GetUserAssignedIdentities(cfg.ResourceGroup, n.Name)
		Expect(err).NotTo(HaveOccurred())
		for resourceID := range *userAssignedIdentities {
			s := strings.Split(resourceID, "/")
			identityName := s[len(s)-1]
			azure.RemoveUserAssignedIdentityFromVM(cfg.ResourceGroup, n.Name, identityName)
		}

		// Remove system assigned identity from all VM
		azure.RemoveSystemAssignedIdentityFromVM(cfg.ResourceGroup, n.Name)
	}
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	AfterEach(func() {
		fmt.Println("\nTearing down the test environment...")

		// Ensure a clean cluster after the end of each test
		cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
		util.PrintCommand(cmd)
		_, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		deleteAllIdentityValidator()
	})

	It("should pass the identity validating test", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)
	})

	It("should not pass the identity validating test if the AzureIdentity is deleted", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		err := azureidentity.DeleteOnCluster(keyvaultIdentity, templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		identityClientID, err := azureidentity.GetClientID(cfg.ResourceGroup, keyvaultIdentity)
		Expect(err).NotTo(HaveOccurred())
		Expect(identityClientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		_, err = validateUserAssignedIdentityOnPod(podName, identityClientID)
		Expect(err).To(HaveOccurred())
	})

	It("should not pass the identity validating test if the AzureIdentityBinding is deleted", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		err := azureidentitybinding.Delete(keyvaultIdentity, templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		identityClientID, err := azureidentity.GetClientID(cfg.ResourceGroup, keyvaultIdentity)
		Expect(err).NotTo(HaveOccurred())
		Expect(identityClientID).NotTo(Equal(""))

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		_, err = validateUserAssignedIdentityOnPod(podName, identityClientID)
		Expect(err).To(HaveOccurred())
	})

	It("should delete the AzureAssignedIdentity if the deployment is deleted", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		waitForDeployDeletion(identityValidator)

		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	It("should establish a new AzureAssignedIdentity and remove the old one when draining the node containing identity validator", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		podName, err := pod.GetNameByPrefix(identityValidator)
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

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		// Make sure the AzureAssignedIdentity is updated along with the new pod
		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

		node.Uncordon(nodeName)
	})

	It("should remove the correct identities when adding AzureIdentity and AzureIdentityBinding in order and removing them in random order", func() {
		testData := make([]struct {
			identityName          string
			identityClientID      string
			identityValidatorName string
			azureAssignedIdentity aadpodid.AzureAssignedIdentity
		}, 5)

		for i := 0; i < 5; i++ {
			identityName := fmt.Sprintf("%s-%d", keyvaultIdentity, i)
			identityValidatorName := fmt.Sprintf("identity-validator-%d", i)

			setUpIdentityAndDeployment(keyvaultIdentity, fmt.Sprintf("%d", i), "1")

			ok, err := azureassignedidentity.WaitOnLengthMatched(i + 1)
			Expect(ok).To(Equal(true))
			Expect(err).NotTo(HaveOccurred())

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
			j := rand.Intn(len(testData))
			testData[i], testData[j] = testData[j], testData[i]
		}

		// Delete i-th elements in test data and check if the identities beyond index i are still functioning
		for i, data := range testData {
			err := azureidentity.DeleteOnCluster(data.identityName, templateOutputPath)
			Expect(err).NotTo(HaveOccurred())

			ok, err := azureassignedidentity.WaitOnLengthMatched(5 - 1 - i)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(Equal(true))

			// Make sure that the identity validator cannot access to the resource group anymore
			_, err = validateUserAssignedIdentityOnPod(data.azureAssignedIdentity.Spec.Pod, data.identityClientID)
			Expect(err).To(HaveOccurred())
			waitForDeployDeletion(data.identityValidatorName)

			// Make sure that the existing identities are still functioning
			for j := i + 1; j < 5; j++ {
				validateAzureAssignedIdentity(testData[j].azureAssignedIdentity, testData[j].identityName)
			}
		}
	})

	// It("should re-schedule the identity validator and its identity to a new node after powering down and restarting the node containing them", func() {
	// 	setUpIdentityAndDeployment(keyvaultIdentity, "")

	// 	identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, keyvaultIdentity)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(identityClientID).NotTo(Equal(""))

	// 	podName, err := pod.GetNameByPrefix(identityValidator)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(podName).NotTo(Equal(""))

	// 	// Get the name of the node to drain
	// 	nodeName, err := pod.GetNodeName(podName)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(nodeName).NotTo(Equal(""))

	// 	azure.StopKubelet(cfg.ResourceGroup, nodeName)
	// 	azure.StopVM(cfg.ResourceGroup, nodeName)

	// 	// 2 AzureAssignedIdentity, one from the powered down node, in Unknown state,
	// 	// and one from the other node, in Running state
	// 	ok, err := azureassignedidentity.WaitOnLengthMatched(2)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(ok).To(Equal(true))

	// 	// Get the new pod name
	// 	podName, err = pod.GetNameByPrefix(identityValidator)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(podName).NotTo(Equal(""))

	// 	azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
	// 	validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

	// 	// Start the VM again to ensure that the old AzureAssignedIdentity is deleted
	// 	azure.StartVM(cfg.ResourceGroup, nodeName)
	// 	azure.StartKubelet(cfg.ResourceGroup, nodeName)

	// 	ok, err = azureassignedidentity.WaitOnLengthMatched(1)
	// 	Expect(ok).To(Equal(true))
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Final validation to ensure that everything is functioning after restarting the node
	// 	azureAssignedIdentity, err = azureassignedidentity.GetByPrefix(identityValidator)
	// 	validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)
	// })

	It("should not alter the user assigned identity on VM after AAD pod identity is created and deleted", func() {
		azureIdentityResource := "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s"
		clusterIdentityResource := fmt.Sprintf(azureIdentityResource, cfg.SubscriptionID, cfg.ResourceGroup, clusterIdentity)
		keyvaultIdentityResource := fmt.Sprintf(azureIdentityResource, cfg.SubscriptionID, cfg.ResourceGroup, keyvaultIdentity)

		// Assign user assigned identity to every node
		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())

		enableUserAssignedIdentityOnCluster(nodeList, clusterIdentity)

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		// Get the name of the node to assign the identity to
		nodeName, err := pod.GetNodeName(podName)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeName).NotTo(Equal(""))

		userAssignedIdentities, err := azure.GetUserAssignedIdentities(cfg.ResourceGroup, nodeName)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(*userAssignedIdentities)).To(Equal(2))

		// Check if both VM identity and pod identity exist in the node
		_, ok := (*userAssignedIdentities)[clusterIdentityResource]
		Expect(ok).To(Equal(true))
		_, ok = (*userAssignedIdentities)[keyvaultIdentityResource]
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

		// Delete pod identity to verify that the VM identity did not get deleted
		waitForDeployDeletion(identityValidator)
		cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
		util.PrintCommand(cmd)
		_, err = cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		ok, err = azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		userAssignedIdentities, err = azure.GetUserAssignedIdentities(cfg.ResourceGroup, nodeName)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(*userAssignedIdentities)).To(Equal(1))

		_, ok = (*userAssignedIdentities)[clusterIdentityResource]
		Expect(ok).To(Equal(true))

		removeUserAssignedIdentityFromCluster(nodeList, clusterIdentity)
	})

	// Require issue #93 to be fixed in order for this test to pass
	// It("should not alter the user assigned identity on VM after assigning and removing the same identity to the pod", func() {
	// 	azureIdentityResource := "/subscriptions/%s/resourceGroups/aad-pod-identity-e2e/providers/Microsoft.ManagedIdentity/userAssignedIdentities/%s"
	// 	clusterIdentityResource := fmt.Sprintf(azureIdentityResource, cfg.SubscriptionID, clusterIdentity)

	// 	// Assign user assigned identity to every node
	// 	nodeList, err := node.GetAll()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Re-deploy aad pod identity infra to allow MIC to register the correct VM level identity
	// 	cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
	// 	_, err = cmd.CombinedOutput()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	enableUserAssignedIdentityOnCluster(nodeList, clusterIdentity)

	// 	cmd = exec.Command("kubectl", "apply", "-f", "../../deploy/infra/deployment-rbac.yaml")
	// 	_, err = cmd.CombinedOutput()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Assign the same identity to identity validator pod
	// 	setUpIdentityAndDeployment(clusterIdentity, "")

	// 	podName, err := pod.GetNameByPrefix(identityValidator)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(podName).NotTo(Equal(""))

	// 	// Get the name of the node to assign the identity to
	// 	nodeName, err := pod.GetNodeName(podName)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(nodeName).NotTo(Equal(""))

	// 	// Make sure that there is only one identity assigned to the VM
	// 	userAssignedIdentities, err := azure.GetUserAssignedIdentities(cfg.ResourceGroup, nodeName)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(len(*userAssignedIdentities)).To(Equal(1))

	// 	_, ok := (*userAssignedIdentities)[clusterIdentityResource]
	// 	Expect(ok).To(Equal(true))

	// 	azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	validateAzureAssignedIdentity(azureAssignedIdentity, clusterIdentity)

	// 	// Delete pod identity to verify that the VM identity did not get deleted
	// 	waitForDeployDeletion(identityValidator)
	// 	cmd = exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding,AzureAssignedIdentity", "--all")
	// 	util.PrintCommand(cmd)
	// 	_, err = cmd.CombinedOutput()
	// 	Expect(err).NotTo(HaveOccurred())

	// 	ok, err = azureassignedidentity.WaitOnLengthMatched(0)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(ok).To(Equal(true))

	// 	userAssignedIdentities, err = azure.GetUserAssignedIdentities(cfg.ResourceGroup, nodeName)
	// 	Expect(err).NotTo(HaveOccurred())
	// 	Expect(len(*userAssignedIdentities)).To(Equal(1))

	// 	_, ok = (*userAssignedIdentities)[clusterIdentityResource]
	// 	Expect(ok).To(Equal(true))

	// 	removeUserAssignedIdentityFromCluster(nodeList, clusterIdentity)
	// })

	It("should cleanup iptable rules after deleting aad-pod-identity", func() {
		// delete the aad-pod-identity deployment
		cmd := exec.Command("kubectl", "delete", "-f", "../../deploy/infra/deployment-rbac.yaml", "--ignore-not-found")
		_, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		ok, err := pod.WaitOnDeletion("nmi")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// check if the iptable rules exist before test
		pods, err := pod.GetAllNameByPrefix("busybox")
		Expect(err).NotTo(HaveOccurred())
		Expect(pods).NotTo(BeNil())

		// check to ensure the custom iptable rules have been cleaned up
		for _, p := range pods {
			// ensure aad-metadata target reference doesn't exist anymore
			out, err := pod.RunCommandInPod("exec", p, "--", "iptables", "-t", "nat", "--check", "PREROUTING", "-j", "aad-metadata")
			Expect(err).To(HaveOccurred())
			Expect(out).To(ContainSubstring("No such file or directory"))

			// ensure aad-metadata custom chain rule doesn't exist anymore
			out, err = pod.RunCommandInPod("exec", p, "--", "iptables", "-t", "nat", "-L", "aad-metadata")
			Expect(err).To(HaveOccurred())
			Expect(out).To(ContainSubstring("No chain/target/match by that name."))
		}

		// reset the infra to previous state
		setupInfra()
	})

	It("should not alter the system assigned identity after creating and deleting pod identity", func() {
		// Assign system assigned identity to every node
		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())

		enableSystemAssignedIdentityOnCluster(nodeList)

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		nodeName, err := pod.GetNodeName(podName)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeName).NotTo(Equal(""))

		// Get the principalID and tenantID of the system assigned identity for verification later
		principalIDBefore, tenantIDBefore, err := azure.GetSystemAssignedIdentity(cfg.ResourceGroup, nodeName)
		Expect(err).NotTo(HaveOccurred())
		Expect(principalIDBefore).NotTo(Equal(""))
		Expect(tenantIDBefore).NotTo(Equal(""))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

		waitForDeployDeletion(identityValidator)
		ok, err := azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// Ensure that the identity is unchanged
		principalIDAfter, tenantIDAfter, err := azure.GetSystemAssignedIdentity(cfg.ResourceGroup, nodeName)
		Expect(err).NotTo(HaveOccurred())
		Expect(principalIDAfter).NotTo(Equal(""))
		Expect(tenantIDAfter).NotTo(Equal(""))
		Expect(principalIDBefore).To(Equal(principalIDAfter))
		Expect(tenantIDAfter).To(Equal(tenantIDAfter))

		removeSystemAssignedIdentityOnCluster(nodeList)
	})

	It("should create azureassignedidentities for 40 pods within ~2mins", func() {
		// setup all the 40 pods in a loop to ensure mic handles
		// scale out efficiently
		for i := 0; i < 5; i++ {
			setUpIdentityAndDeployment(keyvaultIdentity, fmt.Sprintf("%d", i), "8")
		}

		// WaitOnLengthMatched waits for 2 mins, so this will ensure we are performant at high scale
		ok, err := azureassignedidentity.WaitOnLengthMatched(40)
		Expect(ok).To(Equal(true))
		Expect(err).NotTo(HaveOccurred())

		for i := 0; i < 5; i++ {
			identityName := fmt.Sprintf("%s-%d", keyvaultIdentity, i)
			identityValidatorName := fmt.Sprintf("identity-validator-%d", i)

			identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, identityName)
			Expect(err).NotTo(HaveOccurred())
			Expect(identityClientID).NotTo(Equal(""))

			azureAssignedIdentities, err := azureassignedidentity.GetAllByPrefix(identityValidatorName)
			Expect(err).NotTo(HaveOccurred())

			for _, azureAssignedIdentity := range azureAssignedIdentities {
				validateAzureAssignedIdentity(azureAssignedIdentity, identityName)
			}
		}
	})
})

// setupInfra creates the crds, mic, nmi and blocks until iptable entries exist
func setupInfra() {
	// Install CRDs and deploy MIC and NMI
	err := infra.CreateInfra("default", cfg.Registry, cfg.NMIVersion, cfg.MICVersion, templateOutputPath)

	Expect(err).NotTo(HaveOccurred())

	ok, err := daemonset.WaitOnReady(nmiDaemonSet)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))

	// create the helper ssh daemon for validating the iptable rules
	cmd := exec.Command("kubectl", "apply", "-f", "template/busyboxds.yaml")
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	// wait for busbox daemonset to be ready
	ok, err = daemonset.WaitOnReady("busybox")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))

	// check if the iptable rules exist before test
	pods, err := pod.GetAllNameByPrefix("busybox")
	Expect(err).NotTo(HaveOccurred())
	Expect(pods).NotTo(BeNil())

	for _, p := range pods {
		// install iptables in the busybox to keep the vanilla alpine image for busybox
		_, err := pod.RunCommandInPod("exec", p, "--", "apk", "add", "iptables")
		Expect(err).NotTo(HaveOccurred())
	}

	// ensure iptable entries for aad-metadata exist before we begin this test
	// if this fails, then nmi is not working as expected
	ok, err = waitForIPTableRulesToExist("busybox")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
}

// setUpIdentityAndDeployment will deploy AzureIdentity, AzureIdentityBinding, and an identity validator
// Suffix will give the tests the option to add a suffix to the end of the identity name, useful for scale tests
// replicas to indicate the number of replicas for the deployment
func setUpIdentityAndDeployment(azureIdentityName, suffix, replicas string) {
	identityValidatorName := identityValidator

	if suffix != "" {
		azureIdentityName += "-" + suffix
		identityValidatorName += "-" + suffix
	}

	err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	err = azureidentitybinding.Create(azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, cfg.Registry, identityValidatorName, azureIdentityName, cfg.IdentityValidatorVersion, templateOutputPath, replicas)
	Expect(err).NotTo(HaveOccurred())

	ok, err := deploy.WaitOnReady(identityValidatorName)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))

	// additional redundant check to ensure nmi exists and is ready
	// this check is already performed in before suite, so nmi will exist before reaching here
	ok, err = daemonset.WaitOnReady(nmiDaemonSet)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))

	time.Sleep(30 * time.Second)
}

// validateAzureAssignedIdentity will make sure a given AzureAssignedIdentity has the correct properties
func validateAzureAssignedIdentity(azureAssignedIdentity aadpodid.AzureAssignedIdentity, identityName string) {
	identityClientID := azureAssignedIdentity.Spec.AzureIdentityRef.Spec.ClientID
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

	var err error
	var cmdOutput []byte

	if strings.HasPrefix(identityName, keyvaultIdentity) {
		cmdOutput, err = validateUserAssignedIdentityOnPod(podName, identityClientID)
	} else if strings.HasPrefix(identityName, clusterIdentity) {
		cmdOutput, err = validateClusterWideUserAssignedIdentity(podName, identityClientID)
	} else {
		err = errors.Errorf("Invalid identity name: %s", identityName)
	}
	if err != nil {
		fmt.Printf("%s\n", cmdOutput)
	}
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("# %s validated!\n", identityName)
}

// waitForDeployDeletion will block until a give deploy and its pods are completed deleted
func waitForDeployDeletion(deployName string) {
	err := deploy.Delete(deployName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	ok, err := pod.WaitOnDeletion(deployName)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
}

func deleteAllIdentityValidator() {
	cmd := exec.Command("kubectl", "delete", "deploy", "-l", "app="+identityValidator)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	ok, err := pod.WaitOnDeletion(identityValidator)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
}

// validateUserAssignedIdentityOnPod will verify that pod identity is working properly
func validateUserAssignedIdentityOnPod(podName, identityClientID string) ([]byte, error) {
	cmd := exec.Command("kubectl",
		"exec", podName, "--",
		"identityvalidator",
		"--subscription-id", cfg.SubscriptionID,
		"--resource-group", cfg.ResourceGroup,
		"--identity-client-id", identityClientID,
		"--keyvault-name", cfg.KeyvaultName,
		"--keyvault-secret-name", cfg.KeyvaultSecretName,
		"--keyvault-secret-version", cfg.KeyvaultSecretVersion)
	return cmd.CombinedOutput()
}

// validateClusterWideUserAssignedIdentity will verify that VM level identity is working properly
func validateClusterWideUserAssignedIdentity(podName, identityClientID string) ([]byte, error) {
	cmd := exec.Command("kubectl",
		"exec", podName, "--",
		"identityvalidator",
		"--subscription-id", cfg.SubscriptionID,
		"--resource-group", cfg.ResourceGroup,
		"--identity-client-id", identityClientID)
	return cmd.CombinedOutput()
}

// enableUserAssignedIdentityOnCluster will assign an azure identity to all the worker nodes in a cluster
func enableUserAssignedIdentityOnCluster(nodeList *node.List, azureIdentityName string) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}
		err := azure.EnableUserAssignedIdentityOnVM(cfg.ResourceGroup, n.Name, clusterIdentity)
		Expect(err).NotTo(HaveOccurred())
	}
}

// removeUserAssignedIdentityFromCluster will remove an azure identity from all the worker nodes in a cluster
func removeUserAssignedIdentityFromCluster(nodeList *node.List, azureIdentityName string) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}
		err := azure.RemoveUserAssignedIdentityFromVM(cfg.ResourceGroup, n.Name, clusterIdentity)
		Expect(err).NotTo(HaveOccurred())
	}
}

// enableSystemAssignedIdentityOnCluster will enable system assigned identity on the resource group
func enableSystemAssignedIdentityOnCluster(nodeList *node.List) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}
		err := azure.EnableSystemAssignedIdentityOnVM(cfg.ResourceGroup, n.Name)
		Expect(err).NotTo(HaveOccurred())
	}
}

// removeSystemAssignedIdentityOnCluster will remove system assigned identity from the resource group
func removeSystemAssignedIdentityOnCluster(nodeList *node.List) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}
		err := azure.RemoveSystemAssignedIdentityFromVM(cfg.ResourceGroup, n.Name)
		Expect(err).NotTo(HaveOccurred())
	}
}

// waitForIPTableRulesToExist will block until custom chain iptable rules are inserted for each node
func waitForIPTableRulesToExist(prefix string) (bool, error) {
	successChannel, errorChannel := make(chan bool, 1), make(chan error)
	duration, sleep := 120*time.Second, 10*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Println("# Tight-poll to check if iptable rules for aad-metadata exist on all nodes...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChannel <- errors.Errorf("Timeout exceeded (%s) while waiting for iptable rules to exist", duration.String())
			default:
				list, err := pod.GetAllNameByPrefix(prefix)
				if err != nil {
					errorChannel <- err
					return
				}

				found := true
				for _, p := range list {
					// ensure aad-metadata target reference exists
					_, err1 := pod.RunCommandInPod("exec", p, "--", "iptables", "-t", "nat", "--check", "PREROUTING", "-j", "aad-metadata")
					// ensure aad-metadata custom chain rules exists
					_, err2 := pod.RunCommandInPod("exec", p, "--", "iptables", "-t", "nat", "-L", "aad-metadata")
					if err1 != nil || err2 != nil {
						found = false
					}
				}

				if found {
					successChannel <- true
					return
				}

				fmt.Printf("# iptable entries for aad-metadata not found on all nodes. Retrying in %s...\n", sleep.String())
				time.Sleep(sleep)
			}
		}
	}()

	for {
		select {
		case err := <-errorChannel:
			return false, err
		case success := <-successChannel:
			return success, nil
		}
	}
}
