package main

import (
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	cids     = pflag.StringSlice("cids", []string{}, "client ids separated by comma")
	resource = "https://management.azure.com/"
)

func main() {
	pflag.Parse()

	podname := os.Getenv("MY_POD_NAME")
	podnamespace := os.Getenv("MY_POD_NAME")
	podip := os.Getenv("MY_POD_IP")

	log.Info("starting demo pod %s/%s %s", podnamespace, podname, podip)

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
		spt, err := adal.NewServicePrincipalTokenFromMSI(msiEndpoint, resource)
		if err != nil {
			logger.Errorf("Failed to acquire a token using the MSI VM extension, Error: %+v", err)
		}

		for _, userAssignedID := range *cids {
			// Set up the configuration of the service principal
			sptuid, err := adal.NewServicePrincipalTokenFromMSIWithUserAssignedID(msiEndpoint, resource, userAssignedID)
			if err != nil {
				logger.Errorf("Failed to acquire a token using the EMSI VM extension, clientID: %s Error: %+v", userAssignedID, err)
			} else {
				logger.Infof("succesfully acquired a token using the EMSI VM extension, clientID: %s Error: %v", userAssignedID, sptuid.Token().AccessToken)
				if !strings.EqualFold(sptuid.Token().AccessToken, spt.Token().AccessToken) {
					logger.Errorf("MSI and EMSI token endpoints have different access tokens, clientID")
				}
			}
		}
	}
}
