module github.com/Azure/aad-pod-identity/test/e2e

go 1.16

require (
	github.com/Azure/aad-pod-identity v1.6.1
	github.com/Azure/azure-sdk-for-go v40.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.1
	github.com/Azure/go-autorest/autorest/adal v0.9.5
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.12.2
	github.com/onsi/gomega v1.10.1
	github.com/satori/go.uuid v1.2.0 // indirect
	golang.org/x/mod v0.3.0
	k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/client-go v0.19.6
	sigs.k8s.io/controller-runtime v0.6.0
)

replace github.com/Azure/aad-pod-identity => ../..
