package main

import (
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	cids          = pflag.StringSlice("cids", []string{}, "client ids separated by comma")
	retryWaitTime = pflag.Int("retry-wait-time", 20, "retry wait time in seconds")
	resource      = "https://management.azure.com/"
)

func main() {
	pflag.Parse()

	podname := os.Getenv("MY_POD_NAME")
	podnamespace := os.Getenv("MY_POD_NAME")
	podip := os.Getenv("MY_POD_IP")

	log.Infof("starting demo pod %s/%s %s", podnamespace, podname, podip)

	logger := log.WithFields(log.Fields{
		"podnamespace": podnamespace,
		"podname":      podname,
		"podip":        podip,
	})

	msiEndpoint, err := adal.GetMSIVMEndpoint()
	if err != nil {
		logger.Fatalf("failed to get msiendpoint, %+v", err)
	}

	for {
		msitoken := testMSIEndpoint(logger, msiEndpoint, resource)
		if msitoken != nil {
			for _, userAssignedID := range *cids {
				emsitoken := testMSIEndpointFromUserAssignedID(logger, msiEndpoint, userAssignedID, resource)
				if emsitoken != nil {
					if strings.EqualFold(msitoken.AccessToken, emsitoken.AccessToken) {
						logger.Infof("succesfully validated a token using the MSI/EMSI VM extension, msiEndpoint(%s) AccessToken:(%+v)", msiEndpoint, msitoken.AccessToken)
					} else {
						logger.Errorf("mismatch of access tokens using the MSI/EMSI VM extension, msiEndpoint(%s) msi AccessToken:(%+v), emsi AccessToken:(%+v)", msiEndpoint, msitoken.AccessToken, emsitoken.AccessToken)
					}
				}
			}
		}

		time.Sleep(time.Duration(*retryWaitTime) * time.Second)
	}
}

func testMSIEndpoint(logger *log.Entry, msiEndpoint, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
	if err != nil {
		logger.Errorf("Failed to acquire a token using the MSI VM extension, Error: %+v", err)
		return nil
	}
	if err := spt.Refresh(); err != nil {
		logger.Errorf("failed to refresh ServicePrincipalTokenFromMSI using the MSI VM extension,msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("zero token found, MSI VM extension,msiEndpoint(%s)", msiEndpoint)
		return nil
	}
	logger.Infof("succesfully acquired a token using the MSI VM extension,msiEndpoint(%s) Token:(%+v)", msiEndpoint, spt.Token())
	return &token
}

func testMSIEndpointFromUserAssignedID(logger *log.Entry, msiEndpoint, userAssignedID, resource string) *adal.Token {
	spt, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
	if err != nil {
		logger.Errorf("Failed to acquire a token using the EMSI VM extension, clientID: %s Error: %+v", userAssignedID, err)
		return nil
	}
	token := spt.Token()
	if token.IsZero() {
		logger.Errorf("failed to acquire a token using the EMSI VM extension, msiEndpoint(%s) clientID(%s)", msiEndpoint, userAssignedID)
		return nil
	}
	logger.Infof("succesfully acquired a token using the EMSI VM extension, msiEndpoint(%s) clientID(%s) Token:(%+v)", msiEndpoint, userAssignedID, token)
	return &token
}
