// +build e2e_new

package e2e_new

import (
	"testing"

	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azure"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/helm"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/iptables"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/namespace"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	keyvaultIdentity = "keyvault-identity"
)

var (
	clusterProxy      framework.ClusterProxy
	config            *framework.Config
	azureClient       azure.Client
	kubeClient        client.Client
	kubeconfigPath    string
	iptablesNamespace *corev1.Namespace
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "aadpodidentity")
}

var _ = BeforeSuite(func() {
	By("Parsing test configuration")
	var err error
	config, err = framework.ParseConfig()
	Expect(err).To(BeNil())

	By("Creating a Cluster Proxy")
	clusterProxy = framework.NewClusterProxy(initScheme())

	By("Creating an Azure Client")
	azureClient = azure.NewClient(config)

	By("Installing AAD Pod Identity via Helm")
	helm.Install(helm.InstallInput{
		Config: config,
	})

	kubeClient = clusterProxy.GetClient()
	kubeconfigPath = clusterProxy.GetKubeconfigPath()

	iptablesNamespace = namespace.Create(namespace.CreateInput{
		Creator: kubeClient,
		Name:    "iptables",
	})

	iptables.WaitForRules(iptables.WaitForRulesInput{
		Creator:         kubeClient,
		Lister:          kubeClient,
		Namespace:       iptablesNamespace.Name,
		KubeconfigPath:  clusterProxy.GetKubeconfigPath(),
		CreateDaemonSet: true,
		ShouldExist:     true,
	})
})

var _ = AfterSuite(func() {
	By("Dumping logs")

	By("Uninstalling AAD Pod Identity via Helm")
	helm.Uninstall()

	iptables.WaitForRules(iptables.WaitForRulesInput{
		Creator:         kubeClient,
		Lister:          kubeClient,
		Namespace:       iptablesNamespace.Name,
		KubeconfigPath:  clusterProxy.GetKubeconfigPath(),
		CreateDaemonSet: false,
		ShouldExist:     false,
	})

	namespace.Delete(namespace.DeleteInput{
		Deleter:   kubeClient,
		Getter:    kubeClient,
		Namespace: iptablesNamespace,
	})
})

func initScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	framework.TryAddDefaultSchemes(scheme)
	return scheme
}
