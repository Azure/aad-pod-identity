package main

import (
	"flag"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/adal"
	"k8s.io/klog/v2"
)

type assertFunction func() error

const (
	contextTimeout = 80 * time.Second
)

var (
	sleep                 bool
	subscriptionID        string
	identityClientID      string
	identityResourceID    string
	keyvaultName          string
	keyvaultSecretName    string
	keyvaultSecretVersion string
	keyvaultSecretValue   string
)

func init() {
	flag.BoolVar(&sleep, "sleep", false, "Set to true to enter sleep mode")
	flag.StringVar(&subscriptionID, "subscription-id", "", "subscription id for test")
	flag.StringVar(&identityClientID, "identity-client-id", "", "client id for the msi id")
	flag.StringVar(&identityResourceID, "identity-resource-id", "", "resource id for the msi id")
	flag.StringVar(&keyvaultName, "keyvault-name", "", "the name of the keyvault to extract the secret from")
	flag.StringVar(&keyvaultSecretName, "keyvault-secret-name", "", "the name of the keyvault secret we are extracting with pod identity")
	flag.StringVar(&keyvaultSecretVersion, "keyvault-secret-version", "", "the version of the keyvault secret we are extracting with pod identity")
	flag.StringVar(&keyvaultSecretValue, "keyvault-secret-value", "test-value", "the version of the keyvault secret we are extracting with pod identity")
}

func main() {
	flag.Parse()

	if sleep {
		klog.Infof("entering sleep mode")
		for {
			select {}
		}
	}

	podname := os.Getenv("E2E_TEST_POD_NAME")
	podnamespace := os.Getenv("E2E_TEST_POD_NAMESPACE")
	podip := os.Getenv("E2E_TEST_POD_IP")

	klog.Infof("starting identity validator pod %s/%s with pod IP %s", podnamespace, podname, podip)

	imdsTokenEndpoint, _ := adal.GetMSIVMEndpoint()
	kvt := &keyvaultTester{
		client:             keyvault.New(),
		subscriptionID:     subscriptionID,
		identityClientID:   identityClientID,
		identityResourceID: identityResourceID,
		keyvaultName:       keyvaultName,
		secretName:         keyvaultSecretName,
		secretVersion:      keyvaultSecretVersion,
		secretValue:        keyvaultSecretValue,
		imdsTokenEndpoint:  imdsTokenEndpoint,
	}
	spt := &servicePrincipalTester{
		imdsTokenEndpoint: imdsTokenEndpoint,
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	for _, assert := range []assertFunction{
		kvt.assertWithIdentityClientID,
		kvt.assertWithIdentityResourceID,
		spt.assertWithSystemAssignedIdentity,
	} {
		wg.Add(1)
		go func(assert assertFunction) {
			defer wg.Done()
			var err error
			// allow at most 2 retries if we encounter "Identity not found" error
			for i := 0; i < 2; i++ {
				err = assert()
				if !isIdentityNotFoundError(err) {
					break
				}
				if i < 2 {
					time.Sleep(10 * time.Second)
				}
			}
			errCh <- err
		}(assert)
	}
	wg.Wait()

	close(errCh)

	hasError := false
	for err := range errCh {
		if err != nil {
			hasError = true
			klog.Error(err)
		}
	}

	if hasError {
		os.Exit(1)
	}
}

func isIdentityNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Identity not found")
}
