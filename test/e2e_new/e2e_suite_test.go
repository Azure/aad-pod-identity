// +build e2e_new

package e2e_new

import (
	"testing"

	"github.com/Azure/aad-pod-identity/test/e2e_new/framework"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/azure"
	"github.com/Azure/aad-pod-identity/test/e2e_new/framework/helm"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	clusterProxy framework.ClusterProxy
	config       *framework.Config
	azureClient  azure.Client
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
})

var _ = AfterSuite(func() {
	By("Dumping logs")
	By("Uninstalling AAD Pod Identity via Helm")
	helm.Uninstall()
})

func initScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	framework.TryAddDefaultSchemes(scheme)
	return scheme
}
