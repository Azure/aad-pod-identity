module github.com/Azure/aad-pod-identity

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/Azure/azure-sdk-for-go v40.4.0+incompatible
	github.com/Azure/go-autorest/autorest v0.10.0
	github.com/Azure/go-autorest/autorest/adal v0.8.2
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.1.0 // indirect
	github.com/coreos/go-iptables v0.3.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/groupcache v0.0.0-20180513044358-24b0969c4cb7 // indirect
	github.com/google/go-cmp v0.3.0
	github.com/googleapis/gnostic v0.1.0 // indirect
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.0
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	k8s.io/klog v1.0.0
)
