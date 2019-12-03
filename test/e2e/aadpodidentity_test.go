package aadpodidentity

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/Azure/aad-pod-identity/pkg/cloudprovider"
	"github.com/Azure/aad-pod-identity/test/common/azure"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureassignedidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentity"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azureidentitybinding"
	"github.com/Azure/aad-pod-identity/test/common/k8s/azurepodidentityexception"
	"github.com/Azure/aad-pod-identity/test/common/k8s/daemonset"
	"github.com/Azure/aad-pod-identity/test/common/k8s/deploy"
	"github.com/Azure/aad-pod-identity/test/common/k8s/infra"
	"github.com/Azure/aad-pod-identity/test/common/k8s/node"
	"github.com/Azure/aad-pod-identity/test/common/k8s/pod"
	"github.com/Azure/aad-pod-identity/test/common/util"
	"github.com/Azure/aad-pod-identity/test/e2e/config"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	clusterIdentity   = "cluster-identity"
	keyvaultIdentity  = "keyvault-identity"
	identityValidator = "identity-validator"
	nmiDaemonSet      = "nmi"
	immutableIdentity = "immutable-identity"
)

var (
	cfg                config.Config
	templateOutputPath = path.Join("template", "_output")
	logsPath           = path.Join("_output", "logs")
	testIndex          = 0
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
	fmt.Printf("System MSI enabled: %v\n", cfg.SystemMSICluster)
	setupInfra(cfg.Registry, cfg.NMIVersion, cfg.MICVersion, cfg.EnableScaleFeatures, cfg.ImmutableUserMSIs)
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
		m, err := getResourceManager(&n)
		Expect(err).NotTo(HaveOccurred())
		err = m.RemoveAllIdentities()
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = Describe("Kubernetes cluster using aad-pod-identity", func() {
	BeforeEach(func() {
		testIndex++
		fmt.Printf("\n\n[%d] %s\n", testIndex, CurrentGinkgoTestDescription().TestText)
	})

	AfterEach(func() {
		fmt.Println("\nTearing down the test environment...")
		if CurrentGinkgoTestDescription().Failed {
			fmt.Println("Test failed. Collecting debugging information.")
			collectDebuggingInfo()
		}
		// Ensure a clean cluster after the end of each test
		cmd := exec.Command("kubectl", "delete", "AzureIdentity,AzureIdentityBinding", "--all")
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

		n, err := node.Get(nodeName)
		Expect(err).NotTo(HaveOccurred())

		rm, err := getResourceManager(n)
		Expect(err).NotTo(HaveOccurred())

		userAssignedIdentities, err := rm.GetUserAssignedIdentities()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(userAssignedIdentities)).To(Equal(2))

		// Check if both VM identity and pod identity exist in the node
		_, ok := (userAssignedIdentities)[clusterIdentityResource]
		Expect(ok).To(Equal(true))
		_, ok = (userAssignedIdentities)[keyvaultIdentityResource]
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

		userAssignedIdentities, err = rm.GetUserAssignedIdentities()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(userAssignedIdentities)).To(Equal(1))

		_, ok = (userAssignedIdentities)[clusterIdentityResource]
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
		setupInfra(cfg.Registry, cfg.NMIVersion, cfg.MICVersion, cfg.EnableScaleFeatures, cfg.ImmutableUserMSIs)
	})

	It("should not alter the system assigned identity after creating and deleting pod identity", func() {
		if cfg.SystemMSICluster {
			Skip("Test running on system assigned MSI cluster. Skip specific system MSI tests")
		}
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

		n, err := node.Get(nodeName)
		Expect(err).NotTo(HaveOccurred())
		nm, err := getResourceManager(n)
		Expect(err).NotTo(HaveOccurred())

		// Get the principalID and tenantID of the system assigned identity for verification later
		principalIDBefore, tenantIDBefore, err := nm.GetSystemAssignedIdentity()
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
		principalIDAfter, tenantIDAfter, err := nm.GetSystemAssignedIdentity()
		Expect(err).NotTo(HaveOccurred())
		Expect(principalIDAfter).NotTo(Equal(""))
		Expect(tenantIDAfter).NotTo(Equal(""))
		Expect(principalIDBefore).To(Equal(principalIDAfter))
		Expect(tenantIDAfter).To(Equal(tenantIDAfter))

		removeSystemAssignedIdentityOnCluster(nodeList)
	})

	It("should create azureassignedidentities for 40 pods within ~2mins 30seconds", func() {
		// setup all the 40 pods in a loop to ensure mic handles
		// scale out efficiently
		for i := 0; i < 5; i++ {
			setUpIdentityAndDeployment(keyvaultIdentity, fmt.Sprintf("%d", i), "8")
		}

		// WaitOnLengthMatched waits for 2 mins 30 seconds, so this will ensure we are performant at high scale
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

	It("should be backward compatible with old and new version of mic and nmi", func() {
		if cfg.SystemMSICluster {
			Skip("Test running on system assigned MSI cluster. Skip backward compat tests since old versions did not support system MSI clusters")
		}
		// Uninstall CRDs and delete MIC and NMI
		err := deploy.Delete("default", templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := pod.WaitOnDeletion("nmi")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// setup mic and nmi with old releases
		setupInfraOld("mcr.microsoft.com/k8s/aad-pod-identity", "1.4", "1.3", "")

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		ok, err = azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// update the infra to use latest mic and nmi images
		setupInfra(cfg.Registry, cfg.NMIVersion, cfg.MICVersion, cfg.EnableScaleFeatures, cfg.ImmutableUserMSIs)

		ok, err = daemonset.WaitOnReady(nmiDaemonSet)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// TODO (aramase) make this deterministic by ensuring pods with desired image are running
		time.Sleep(30 * time.Second)

		ok, err = waitForIPTableRulesToExist("busybox")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)
	})

	It("should not delete an in use identity from a vmss", func() {
		// This test is specifically testing vmss behavior
		// As such we'll look through the cluster to see if there are nodes assigned
		// to a vmss, and if any of thoe vmss's have more than one node.
		//
		// We cannot do the test if there is not at least1 vmss with at least 2 nodes
		// attach to it.
		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())

		vmss, vmssID := GetVMSS(nodeList)
		if vmssID == "" {
			Skip("Skipping test since there is no vmss with more than 1 node")
			return
		}

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1", func(d *infra.IdentityValidatorTemplateData) {
			d.NodeName = vmss[0].Name
		})
		defer waitForDeployDeletion(identityValidator)

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		nodeName, err := pod.GetNodeName(podName)
		Expect(err).NotTo(HaveOccurred())
		Expect(nodeName).NotTo(Equal(""))

		data := infra.IdentityValidatorTemplateData{
			Name:                     identityValidator + "2",
			IdentityBinding:          keyvaultIdentity,
			Registry:                 cfg.Registry,
			IdentityValidatorVersion: cfg.IdentityValidatorVersion,
			NodeName:                 vmss[1].Name,
		}

		err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, templateOutputPath, data)
		Expect(err).NotTo(HaveOccurred())

		exists, err := azure.UserIdentityAssignedToVMSS(cfg.ResourceGroup, vmssID, keyvaultIdentity)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(Equal(true))

		waitForDeployDeletion(identityValidator + "2")

		// TODO: this is racey, there is no way to know if MIC has even done a reconciliation
		exists, err = azure.UserIdentityAssignedToVMSS(cfg.ResourceGroup, vmssID, keyvaultIdentity)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(Equal(true))
	})

	It("should not delete the Immutable Identity from vmss when the deployment is deleted", func() {
		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())

		vmss, vmssID := GetVMSS(nodeList)
		if vmssID == "" {
			Skip("Skipping test since there is no vmss with more than 1 node")
			return
		}

		// Explicitly assign identity to the underlying VMSS:
		enableUserAssignedIdentityOnCluster(nodeList, immutableIdentity)

		setUpIdentityAndDeployment(immutableIdentity, "", "1", func(d *infra.IdentityValidatorTemplateData) {
			d.NodeName = vmss[0].Name
		})

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		// Ensure that the identity validator is able to get the token
		validateAzureAssignedIdentity(azureAssignedIdentity, immutableIdentity)

		waitForDeployDeletion(identityValidator)

		ok, err = azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		exists, err := azure.UserIdentityAssignedToVMSS(cfg.ResourceGroup, vmssID, immutableIdentity)
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(Equal(true))

	})

	It("should pass liveness probe test", func() {
		pods, err := pod.GetAllNameByPrefix("mic")
		Expect(err).NotTo(HaveOccurred())
		Expect(pods).NotTo(BeNil())

		leader, err := getMICLeader()
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("MIC leader: %s\n", leader)

		for _, p := range pods {
			// Leader MIC will show as active and other as Not Active
			if strings.EqualFold(p, leader) {
				checkHealthProbe(p, "Active")
			} else {
				checkHealthProbe(p, "Not Active")
			}
		}

		pods, err = pod.GetAllNameByPrefix("nmi")
		Expect(err).NotTo(HaveOccurred())
		Expect(pods).NotTo(BeNil())
		for _, p := range pods {
			checkHealthProbe(p, "Active")
		}
	})

	It("should pass validation by bypassing nmi using azurepodidentityexception crd", func() {
		// Creates 2 pods with labels defined in AzurePodIdentityException
		// Creates 1 pod that needs to go through pod-identity
		// Validates the mixed scenario works as expected.

		// Assign user assigned identity to every node
		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())

		enableUserAssignedIdentityOnCluster(nodeList, fmt.Sprintf("%s-%d", keyvaultIdentity, 1))
		enableUserAssignedIdentityOnCluster(nodeList, fmt.Sprintf("%s-%d", keyvaultIdentity, 2))

		// If we have system assigned MSI cluster, the identity validator check for generating system
		// assigned MSI token will work without adding system assigned identity explicitly.
		if !cfg.SystemMSICluster {
			enableSystemAssignedIdentityOnCluster(nodeList)
		}

		err = azurepodidentityexception.Create(fmt.Sprintf("%s-%d", identityValidator, 1), templateOutputPath, map[string]string{"thispod": "shouldexcept"})
		Expect(err).NotTo(HaveOccurred())

		err = azurepodidentityexception.Create(fmt.Sprintf("%s-%d", identityValidator, 2), templateOutputPath, map[string]string{"thispod": "alsoshouldexcept"})
		Expect(err).NotTo(HaveOccurred())

		data := infra.IdentityValidatorTemplateData{
			Name:                     fmt.Sprintf("%s-%d", identityValidator, 1),
			IdentityBinding:          "random",
			Registry:                 cfg.Registry,
			IdentityValidatorVersion: cfg.IdentityValidatorVersion,
			Replicas:                 "1",
			DeploymentLabels:         map[string]string{"thispod": "shouldexcept"},
		}

		err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, templateOutputPath, data)
		Expect(err).NotTo(HaveOccurred())

		data = infra.IdentityValidatorTemplateData{
			Name:                     fmt.Sprintf("%s-%d", identityValidator, 2),
			IdentityBinding:          "random",
			Registry:                 cfg.Registry,
			IdentityValidatorVersion: cfg.IdentityValidatorVersion,
			Replicas:                 "1",
			DeploymentLabels:         map[string]string{"thispod": "alsoshouldexcept"},
		}

		err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, templateOutputPath, data)
		Expect(err).NotTo(HaveOccurred())

		ok, err := deploy.WaitOnReady(fmt.Sprintf("%s-%d", identityValidator, 1))
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		ok, err = deploy.WaitOnReady(fmt.Sprintf("%s-%d", identityValidator, 2))
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		setUpIdentityAndDeployment(keyvaultIdentity, "0", "1")

		ok, err = azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		// This pod should go through the nmi as it has the aadpodidbinding label and doesn't contain exception crd
		podName0, err := pod.GetNameByPrefix(fmt.Sprintf("%s-%d", identityValidator, 0))
		Expect(err).NotTo(HaveOccurred())
		Expect(podName0).NotTo(Equal(""))

		identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, fmt.Sprintf("%s-%d", keyvaultIdentity, 0))
		Expect(err).NotTo(HaveOccurred())

		cmdOutput, err := validateUserAssignedIdentityOnPod(podName0, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())

		// Pod1 and Pod2 have labels matching labels defined in exception crds. So NMI should proxy the request as is
		// and send the token back without any validation.
		podName1, err := pod.GetNameByPrefix(fmt.Sprintf("%s-%d", identityValidator, 1))
		Expect(err).NotTo(HaveOccurred())
		Expect(podName1).NotTo(Equal(""))

		identityClientID, err = azure.GetIdentityClientID(cfg.ResourceGroup, fmt.Sprintf("%s-%d", keyvaultIdentity, 1))
		Expect(err).NotTo(HaveOccurred())

		cmdOutput, err = validateUserAssignedIdentityOnPod(podName1, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())

		podName2, err := pod.GetNameByPrefix(fmt.Sprintf("%s-%d", identityValidator, 2))
		Expect(err).NotTo(HaveOccurred())
		Expect(podName2).NotTo(Equal(""))

		identityClientID, err = azure.GetIdentityClientID(cfg.ResourceGroup, fmt.Sprintf("%s-%d", keyvaultIdentity, 2))
		Expect(err).NotTo(HaveOccurred())

		cmdOutput, err = validateUserAssignedIdentityOnPod(podName2, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())

		removeUserAssignedIdentityFromCluster(nodeList, fmt.Sprintf("%s-%d", keyvaultIdentity, 1))
		removeUserAssignedIdentityFromCluster(nodeList, fmt.Sprintf("%s-%d", keyvaultIdentity, 2))
		if !cfg.SystemMSICluster {
			removeSystemAssignedIdentityOnCluster(nodeList)
		}
	})

	It("should pass identity validation with correct identity and fail with wrong identity", func() {
		// This test is to ensure when 2 identities for the pod exist, the
		// correct identity is used based on the client id in the request.
		// keyvault-identity has the right permissions to get and list secret
		// keyvault-identity-5 only has permission to list and should fail to get secret.

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, fmt.Sprintf("%s-%d", keyvaultIdentity, 5), templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		err = azureidentitybinding.Create(fmt.Sprintf("%s-%d", keyvaultIdentity, 5), keyvaultIdentity, templateOutputPath)
		Expect(err).NotTo(HaveOccurred())

		ok, err := azureassignedidentity.WaitOnLengthMatched(2)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		podName, err := pod.GetNameByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(podName).NotTo(Equal(""))

		identityClientID, err := azure.GetIdentityClientID(cfg.ResourceGroup, fmt.Sprintf("%s-%d", keyvaultIdentity, 5))
		Expect(err).NotTo(HaveOccurred())

		cmdOutput, err := validateUserAssignedIdentityOnPod(podName, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).To(HaveOccurred())

		identityClientID, err = azure.GetIdentityClientID(cfg.ResourceGroup, keyvaultIdentity)
		Expect(err).NotTo(HaveOccurred())

		cmdOutput, err = validateUserAssignedIdentityOnPod(podName, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())
	})

	It("should delete assigned identity when identity no longer exists on underlying node", func() {
		setUpIdentityAndDeployment(keyvaultIdentity, "", "1")

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

		nodeList, err := node.GetAll()
		Expect(err).NotTo(HaveOccurred())
		// remove the assigned identity manually from the underlying node
		removeUserAssignedIdentityFromCluster(nodeList, keyvaultIdentity)

		waitForDeployDeletion(identityValidator)
		ok, err = azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	})

	It("should pass multiple identity validating test even when MIC is failing over", func() {
		// Two go routines - one which keeps assigning identities by means of identity validator assignment.
		iterations := 2
		var wg sync.WaitGroup
		wg.Add(2)

		go func(iterations int) {
			defer wg.Done()
			runMICDisrupt(iterations)
			fmt.Printf("Disrupting MIC completed %d iterations\n", iterations)
		}(iterations)

		go func(iterations int) {
			defer wg.Done()
			runValidatorTest(iterations)
			fmt.Printf("Validator tests completed %d iterations\n", iterations)
		}(iterations)

		wg.Wait()
		fmt.Printf("Done with running validator test and disrupting MIC")
	})

	It("should assign identity with init containers", func() {
		// should create an assigned identity when pod with init container is created
		// In this test, we run az login --identity in an init container. This command will succeed only when
		// identity has been successfully assigned for pod by NMI. Then we also perform an additional validation
		// for the user assigned identity within the identity validator container.

		setUpIdentityAndDeployment(keyvaultIdentity, "", "1", func(d *infra.IdentityValidatorTemplateData) {
			d.InitContainer = true
		})

		ok, err := azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())

		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)
	})

	It("should pass the identity format validation with gatekeeper constraint", func() {

		// setup the required infra
		setupIdentityFormatValidationInfra()

		// cleanup the infra created for this specific test
		// uninstall the identity format constraint ,identity format template,Gatekeeper in sequence
		defer cleanupIdentityFormatValidationInfra()

		cmd := exec.Command("kubectl", "apply", "-f", "template/aadpodidentity_test_invalid.yaml")
		util.PrintCommand(cmd)
		output, err := cmd.CombinedOutput()
		fmt.Printf("%s", output)
		// this should fail given the constraint on the resourceId format
		Expect(err).To(HaveOccurred())

		cmd = exec.Command("kubectl", "apply", "-f", "template/aadpodidentity_test_valid.yaml")
		util.PrintCommand(cmd)
		output, err = cmd.CombinedOutput()
		fmt.Printf("%s", output)
		Expect(err).NotTo(HaveOccurred())
	})
})

func GetVMSS(nodeList *node.List) ([]node.Node, string) {
	vmssNodes := make(map[string][]node.Node)
	for _, n := range nodeList.Nodes {
		r, _ := cloudprovider.ParseResourceID(n.Spec.ProviderID)
		if r.ResourceType == cloudprovider.VMSSResourceType {
			ls := vmssNodes[r.ResourceName]
			vmssNodes[r.ResourceName] = append(ls, n)
		}
	}
	var vmssID string
	for id, ls := range vmssNodes {
		if len(ls) > 1 {
			vmssID = id
			break
		}
	}
	if vmssID == "" {
		return nil, ""
	}
	vmss := vmssNodes[vmssID]
	return vmss, vmssID
}

func runValidatorTest(iterations int) {
	defer GinkgoRecover()
	replicas := "1"
	data := infra.IdentityValidatorTemplateData{
		Name:                     identityValidator,
		IdentityBinding:          keyvaultIdentity,
		Registry:                 cfg.Registry,
		IdentityValidatorVersion: cfg.IdentityValidatorVersion,
		Replicas:                 replicas,
	}
	var err error

	for i := 0; i < iterations; i++ {
		fmt.Printf("Starting identity validator. Iteration: %d\n", i)

		if i == 0 {
			// Initial creation of identity, binding and identityvalidator pod.
			setUpIdentityAndDeployment(keyvaultIdentity, "", "1")
		} else { // After the initial one only create and delete the identity validator.
			err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, templateOutputPath, data)
			Expect(err).NotTo(HaveOccurred())
		}

		ok, err := deploy.WaitOnReady(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		ok, err = azureassignedidentity.WaitOnLengthMatched(1)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))

		azureAssignedIdentity, err := azureassignedidentity.GetByPrefix(identityValidator)
		Expect(err).NotTo(HaveOccurred())
		validateAzureAssignedIdentity(azureAssignedIdentity, keyvaultIdentity)

		deleteAllIdentityValidator()

		ok, err = azureassignedidentity.WaitOnLengthMatched(0)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(Equal(true))
	}
	fmt.Printf("Completing %d validation checks\n", iterations)
}

func runMICDisrupt(iterations int) {
	defer GinkgoRecover()
	delay := time.Second * 60
	for i := 0; i < iterations; i++ {
		fmt.Printf("Starting MIC disruptor. Iteration: %d\n", i)
		leader, err := getMICLeader()
		Expect(err).NotTo(HaveOccurred())
		fmt.Printf("MIC leader: %s\n", leader)
		Expect(pod.DeletePod(leader)).NotTo(HaveOccurred())
		waitForLeaderChange(leader)
		// Wait for some time for MIC to go through some iterations.
		time.Sleep(delay)
	}
	fmt.Printf("Completing disrupting MIC %d times.\n", iterations)
}

func waitForLeaderChange(checkLeader string) {
	// Total 60 seconds for leader change.
	retries := 12
	sleepTime := time.Second * 5

	for i := 0; i < retries; i++ {
		currentLeader, err := getMICLeader()
		Expect(err).NotTo(HaveOccurred())
		if !strings.EqualFold(currentLeader, checkLeader) {
			fmt.Printf("Leader changed from %s to %s\n", checkLeader, currentLeader)
			return
		}
		time.Sleep(sleepTime)
	}
	Expect(false).Should(Equal(true), "Leader change did not happen in 60 seconds")
}

func collectLogs(podName, dir string) {
	pods, err := pod.GetAllNameByPrefix(podName)
	Expect(err).NotTo(HaveOccurred())
	for _, p := range pods {
		logFile := path.Join(dir, p)
		cmd := exec.Command("bash", "-c", "kubectl logs "+p+" > "+logFile)
		util.PrintCommand(cmd)
		_, err := cmd.Output()
		Expect(err).NotTo(HaveOccurred())
	}
}

func collectPods(dir string) {
	logFile := path.Join(dir, "pods")
	cmd := exec.Command("bash", "-c", "kubectl get pods -o wide > "+logFile)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func collectEvents(dir string) {
	logFile := path.Join(dir, "events")
	cmd := exec.Command("bash", "-c", "kubectl get  events --sort-by='.metadata.creationTimestamp' >"+logFile)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func collectLeEndpoint(dir string) {
	logFile := path.Join(dir, "leEndpoint")
	cmd := exec.Command("bash", "-c", "kubectl get endpoints aad-pod-identity-mic -o yaml >"+logFile)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func collectAadpodidentityInfoInternal(dir, crdName string) {
	logFile := path.Join(dir, crdName)
	cmd := exec.Command("bash", "-c", "kubectl get "+crdName+" -o yaml >"+logFile)
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())
}

func collectAadpodidentityInfo(dir string) {
	collectAadpodidentityInfoInternal(dir, "azureidentities")
	collectAadpodidentityInfoInternal(dir, "azureidentitybindings")
	collectAadpodidentityInfoInternal(dir, "azureassignedidentities")
}

func collectDebuggingInfo() {
	tNow := time.Now()
	logDirName := path.Join(logsPath, tNow.Format(time.RFC3339))
	err := os.MkdirAll(logDirName, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	infoFile := path.Join(logDirName, "info")
	fd, err := os.Create(infoFile)
	Expect(err).NotTo(HaveOccurred())

	fd.WriteString("Test name: " + CurrentGinkgoTestDescription().TestText + "\n")
	fd.WriteString("Collecting diagnostics at: " + tNow.Format(time.UnixDate))
	fd.Sync()
	fd.Close()

	collectPods(logDirName)
	collectEvents(logDirName)
	collectLeEndpoint(logDirName)
	collectAadpodidentityInfo(logDirName)
	collectLogs("mic", logDirName)
	collectLogs("nmi", logDirName)
	collectLogs("identityvalidator", logDirName)
	collectLogs("busybox", logDirName)
}

func getMICLeader() (string, error) {
	cmd := exec.Command("kubectl", "get", "endpoints", "aad-pod-identity-mic", "-o", "json")
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	const leAnnotation = "control-plane.alpha.kubernetes.io/leader"

	ep := &corev1.Endpoints{}
	if err := json.Unmarshal(output, &ep); err != nil {
		return "", errors.Wrap(err, "Failed to unmarshall json")
	}

	leRecordStr := ep.Annotations[leAnnotation]
	if leRecordStr == "" {
		return "", fmt.Errorf("Leader election record empty ")
	}

	leRecord := &rl.LeaderElectionRecord{}
	if err := json.Unmarshal([]byte(leRecordStr), leRecord); err != nil {
		return "", errors.Wrapf(err, "Could not unmarshall: %s. Error: %+v", leRecordStr, err)
	}
	return leRecord.HolderIdentity, nil
}

func checkProbe(p string, endpoint string) string {
	output, err := pod.RunCommandInPod("exec", p, "--", "wget", "http://127.0.0.1:8080/"+endpoint, "-q", "-O", "-")
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("Output: %s\n", output)
	return output
}

func checkHealthProbe(p string, state string) {
	Expect(strings.EqualFold(state, checkProbe(p, "healthz"))).To(BeTrue())
}

func checkInfra() {
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

// setupInfra creates the crds, mic, nmi and blocks until iptable entries exist
func setupInfraOld(registry, nmiVersion, micVersion string, immutableUserMSIs string) {
	// Install CRDs and deploy MIC and NMI
	err := infra.CreateInfra("default", registry, nmiVersion, micVersion, templateOutputPath, true, false, immutableUserMSIs)
	Expect(err).NotTo(HaveOccurred())
	checkInfra()
}

// setupInfra creates the crds, mic, nmi and blocks until iptable entries exist
func setupInfra(registry, nmiVersion, micVersion string, enableScaleFeatures bool, immutableUserMSIs string) {
	// Install CRDs and deploy MIC and NMI
	err := infra.CreateInfra("default", registry, nmiVersion, micVersion, templateOutputPath, false, enableScaleFeatures, immutableUserMSIs)
	Expect(err).NotTo(HaveOccurred())
	checkInfra()
}

// setUpIdentityAndDeployment will deploy AzureIdentity, AzureIdentityBinding, and an identity validator
// Suffix will give the tests the option to add a suffix to the end of the identity name, useful for scale tests
// replicas to indicate the number of replicas for the deployment
func setUpIdentityAndDeployment(azureIdentityName, suffix, replicas string, tmplOpts ...func(*infra.IdentityValidatorTemplateData)) {
	identityValidatorName := identityValidator

	if suffix != "" {
		azureIdentityName += "-" + suffix
		identityValidatorName += "-" + suffix
	}

	err := azureidentity.CreateOnCluster(cfg.SubscriptionID, cfg.ResourceGroup, azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	err = azureidentitybinding.Create(azureIdentityName, azureIdentityName, templateOutputPath)
	Expect(err).NotTo(HaveOccurred())

	data := infra.IdentityValidatorTemplateData{
		Name:                     identityValidatorName,
		IdentityBinding:          azureIdentityName,
		Registry:                 cfg.Registry,
		IdentityValidatorVersion: cfg.IdentityValidatorVersion,
		Replicas:                 replicas,
	}

	for _, o := range tmplOpts {
		o(&data)
	}

	err = infra.CreateIdentityValidator(cfg.SubscriptionID, cfg.ResourceGroup, templateOutputPath, data)
	Expect(err).NotTo(HaveOccurred())

	ok, err := deploy.WaitOnReady(identityValidatorName)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))

	// additional redundant check to ensure nmi exists and is ready
	// this check is already performed in before suite, so nmi will exist before reaching here
	ok, err = daemonset.WaitOnReady(nmiDaemonSet)
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(Equal(true))
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

	if strings.HasPrefix(identityName, keyvaultIdentity) {
		cmdOutput, err := validateUserAssignedIdentityOnPod(podName, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())
	} else if strings.HasPrefix(identityName, clusterIdentity) {
		cmdOutput, err := validateClusterWideUserAssignedIdentity(podName, identityClientID)
		Expect(errors.Wrap(err, string(cmdOutput))).NotTo(HaveOccurred())
	} else {
		err := errors.Errorf("Invalid identity name: %s", identityName)
		Expect(err).NotTo(HaveOccurred())
	}

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
	util.PrintCommand(cmd)
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
	util.PrintCommand(cmd)
	return cmd.CombinedOutput()
}

// enableUserAssignedIdentityOnCluster will assign an azure identity to all the worker nodes in a cluster
func enableUserAssignedIdentityOnCluster(nodeList *node.List, identityName string) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}

		// TODO: this should be optimized for vmss
		m, err := getResourceManager(&n)
		Expect(err).NotTo(HaveOccurred())

		err = m.EnableUserAssignedIdentity(identityName)
		Expect(err).NotTo(HaveOccurred())
	}
}

// removeUserAssignedIdentityFromCluster will remove an azure identity from all the worker nodes in a cluster
func removeUserAssignedIdentityFromCluster(nodeList *node.List, identityName string) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}

		// TODO: this should be optimized for vmss
		m, err := getResourceManager(&n)
		Expect(err).NotTo(HaveOccurred())

		err = m.RemoveUserAssignedIdentity(identityName)
		Expect(err).NotTo(HaveOccurred())
	}
}

// enableSystemAssignedIdentityOnCluster will enable system assigned identity on the resource group
func enableSystemAssignedIdentityOnCluster(nodeList *node.List) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}

		// TODO: this should be optimized for vmss
		m, err := getResourceManager(&n)
		Expect(err).NotTo(HaveOccurred())

		err = m.EnableSystemAssignedIdentity()
		Expect(err).NotTo(HaveOccurred())
	}
}

// removeSystemAssignedIdentityOnCluster will remove system assigned identity from the resource group
func removeSystemAssignedIdentityOnCluster(nodeList *node.List) {
	for _, n := range nodeList.Nodes {
		if strings.Contains(n.Name, "master") {
			continue
		}

		// TODO: this should be optimized for vmss
		nm, err := getResourceManager(&n)
		Expect(err).NotTo(HaveOccurred())

		err = nm.RemoveSystemAssignedIdentity()
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

func getResourceManager(n *node.Node) (resourceManager, error) {
	r, err := cloudprovider.ParseResourceID(n.Spec.ProviderID)
	if err != nil {
		return nil, err
	}

	switch r.ResourceType {
	case cloudprovider.VMResourceType:
		return vmManager(r), nil
	case cloudprovider.VMSSResourceType:
		return vmssManager(r), nil
	default:
		panic("unknown resource type: %s" + r.ResourceType)
	}
}

// setupIdentityFormatValidationInfra install Gatekeeper, format template and constraints
func setupIdentityFormatValidationInfra() {
	// install Gatekeeper policy controller
	err := infra.InstallGatekeeper()
	Expect(err).NotTo(HaveOccurred())

	var output []byte
	// install identity format template
	cmd := exec.Command("kubectl", "apply", "-f", "../../validation/gatekeeper/azureidentityformat_template.yaml")
	util.PrintCommand(cmd)
	output, err = cmd.CombinedOutput()
	fmt.Printf("%s", output)
	Expect(err).NotTo(HaveOccurred())

	// constraint template takes time to init to handle request, leading to failure
	// added to make reliable, can be converted to deterministic sleep by retrying after GET on expected resource
	time.Sleep(60 * time.Second)

	// install identity format constraint
	cmd = exec.Command("kubectl", "apply", "-f", "../../validation/gatekeeper/azureidentityformat_constraint.yaml")
	util.PrintCommand(cmd)
	output, err = cmd.CombinedOutput()
	fmt.Printf("%s", output)
	Expect(err).NotTo(HaveOccurred())

	// constraint takes time to init
	time.Sleep(60 * time.Second)
}

// cleanupIdentityFormatValidationInfra delete Gatekeeper, format template and constraints
func cleanupIdentityFormatValidationInfra() {

	// uninstall identity format constraint
	cmd := exec.Command("kubectl", "delete", "-f", "../../validation/gatekeeper/azureidentityformat_constraint.yaml")
	util.PrintCommand(cmd)
	_, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	// uninstall identity format template
	cmd = exec.Command("kubectl", "delete", "-f", "../../validation/gatekeeper/azureidentityformat_template.yaml")
	util.PrintCommand(cmd)
	_, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	// uninstall Gatekeeper policy controller
	err = infra.UninstallGatekeeper()
	Expect(err).NotTo(HaveOccurred())
}

type resourceManager interface {
	GetUserAssignedIdentities() (map[string]azure.UserAssignedIdentity, error)
	GetSystemAssignedIdentity() (string, string, error)
	RemoveAllIdentities() error
	RemoveUserAssignedIdentity(id string) error
	RemoveSystemAssignedIdentity() error
	EnableSystemAssignedIdentity() error
	EnableUserAssignedIdentity(id string) error
}

type vmManager azure.Resource

func (m vmManager) GetUserAssignedIdentities() (map[string]azure.UserAssignedIdentity, error) {
	return azure.GetVMUserAssignedIdentities(m.ResourceGroup, m.ResourceName)
}

func (m vmManager) GetSystemAssignedIdentity() (string, string, error) {
	return azure.GetVMSystemAssignedIdentity(m.ResourceGroup, m.ResourceName)
}

func (m vmManager) RemoveUserAssignedIdentity(id string) error {
	return azure.RemoveUserAssignedIdentityFromVM(m.ResourceGroup, m.ResourceName, id)
}

func (m vmManager) RemoveSystemAssignedIdentity() error {
	return azure.RemoveSystemAssignedIdentityFromVM(m.ResourceGroup, m.ResourceName)
}

type errList []error

func (ls errList) Error() string {
	var buf strings.Builder

	for i, err := range ls {
		s := err.Error()
		if i < len(ls)-1 {
			s += "\n"
		}
		buf.WriteString(s)
	}
	return buf.String()
}

func (m vmManager) RemoveAllIdentities() error {
	uIDs, err := m.GetUserAssignedIdentities()
	if err != nil {
		return err
	}

	var errs errList

	for resourceID := range uIDs {
		s := strings.Split(resourceID, "/")
		id := s[len(s)-1]
		err := m.RemoveUserAssignedIdentity(id)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if !cfg.SystemMSICluster {
		if err := m.RemoveSystemAssignedIdentity(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (m vmManager) EnableSystemAssignedIdentity() error {
	return azure.EnableSystemAssignedIdentityOnVM(m.ResourceGroup, m.ResourceName)
}

func (m vmManager) EnableUserAssignedIdentity(id string) error {
	return azure.EnableUserAssignedIdentityOnVM(m.ResourceGroup, m.ResourceName, id)
}

type vmssManager azure.Resource

func (m vmssManager) GetUserAssignedIdentities() (map[string]azure.UserAssignedIdentity, error) {
	return azure.GetVMSSUserAssignedIdentities(m.ResourceGroup, m.ResourceName)
}

func (m vmssManager) GetSystemAssignedIdentity() (string, string, error) {
	return azure.GetVMSSSystemAssignedIdentity(m.ResourceGroup, m.ResourceName)
}

func (m vmssManager) RemoveUserAssignedIdentity(id string) error {
	return azure.RemoveUserAssignedIdentityFromVMSS(m.ResourceGroup, m.ResourceName, id)
}

func (m vmssManager) RemoveSystemAssignedIdentity() error {
	return azure.RemoveSystemAssignedIdentityFromVMSS(m.ResourceGroup, m.ResourceName)
}

func (m vmssManager) RemoveAllIdentities() error {
	uIDs, err := m.GetUserAssignedIdentities()
	if err != nil {
		return err
	}

	var errs errList

	for resourceID := range uIDs {
		s := strings.Split(resourceID, "/")
		id := s[len(s)-1]
		err := m.RemoveUserAssignedIdentity(id)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if !cfg.SystemMSICluster {
		if err := m.RemoveSystemAssignedIdentity(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (m vmssManager) EnableSystemAssignedIdentity() error {
	return azure.EnableSystemAssignedIdentityOnVMSS(m.ResourceGroup, m.ResourceName)
}

func (m vmssManager) EnableUserAssignedIdentity(id string) error {
	return azure.EnableUserAssignedIdentityOnVMSS(m.ResourceGroup, m.ResourceName, id)
}
