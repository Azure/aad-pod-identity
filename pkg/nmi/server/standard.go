package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	utils "github.com/Azure/aad-pod-identity/pkg/utils"
	"github.com/Azure/go-autorest/autorest/adal"

	log "github.com/sirupsen/logrus"
)

// StandardClient ...
type StandardClient struct {
	TokenClient
}

// NewStandardTokenClient creates new standard nmi client
func NewStandardTokenClient() *StandardClient {
	return &StandardClient{}
}

// GetIdentities ...
func (s *Server) GetIdentities(ctx context.Context, podns, podname, rqClientID string) ([]aadpodid.AzureIdentity, bool, error) {
	attempt := 0
	var err error
	var idStateMap map[string][]aadpodid.AzureIdentity

	// this loop will run to ensure we have assigned identities before we return. If there are no assigned identities in created state within 80s (16 retries * 5s wait) then we return an error.
	// If we get an assigned identity in created state within 80s, then loop will continue until 100s to find assigned identity in assigned state.
	// Retry interval for CREATED state is set to 80s because avg time for identity to be assigned to the node is 35-37s.
	for attempt < s.ListPodIDsRetryAttemptsForCreated+s.ListPodIDsRetryAttemptsForAssigned {
		idStateMap, err = s.KubeClient.ListPodIds(podns, podname)
		if err == nil {
			if len(rqClientID) == 0 {
				// check to ensure backward compatability with assignedIDs that have no state
				// assigned identites created with old version of mic will not contain a state. So first we check to see if an assigned identity with
				// no state exists that matches req client id.
				if len(idStateMap[""]) != 0 {
					log.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
					return idStateMap[""], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDAssigned]) != 0 {
					return idStateMap[aadpodid.AssignedIDAssigned], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDCreated]) == 0 && attempt >= s.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, s.ListPodIDsRetryAttemptsForCreated, s.ListPodIDsRetryIntervalInSeconds, err)
				}
			} else {
				// if client id exists in request, we need to ensure the identity with this client
				// exists and is in Assigned state
				// check to ensure backward compatability with assignedIDs that have no state
				for _, podID := range idStateMap[""] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						log.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
						return idStateMap[""], true, nil
					}
				}
				for _, podID := range idStateMap[aadpodid.AssignedIDAssigned] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						return idStateMap[aadpodid.AssignedIDAssigned], true, nil
					}
				}
				var foundMatch bool
				for _, podID := range idStateMap[aadpodid.AssignedIDCreated] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						foundMatch = true
						break
					}
				}
				if !foundMatch && attempt >= s.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, s.ListPodIDsRetryAttemptsForCreated, s.ListPodIDsRetryIntervalInSeconds, err)
				}
			}
		}
		attempt++

		select {
		case <-time.After(time.Duration(s.ListPodIDsRetryIntervalInSeconds) * time.Second):
		case <-ctx.Done():
			err = ctx.Err()
			return nil, true, err
		}
		log.Debugf("failed to get assigned ids for pod:%s/%s in ASSIGNED state, retrying attempt: %d", podns, podname, attempt)
	}
	return nil, true, fmt.Errorf("getting assigned identities for pod %s/%s in ASSIGNED state failed after %d attempts, retry duration [%d]s. Error: %v",
		podns, podname, s.ListPodIDsRetryAttemptsForCreated+s.ListPodIDsRetryAttemptsForAssigned, s.ListPodIDsRetryIntervalInSeconds, err)
}

// GetToken ...
func (s *Server) GetToken(ctx context.Context, rqClientID, rqResource string, podIDs []aadpodid.AzureIdentity) (token *adal.Token, clientID string, err error) {
	rqHasClientID := len(rqClientID) != 0
	for _, v := range podIDs {
		clientID := v.Spec.ClientID
		if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
			log.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
			continue
		}

		idType := v.Spec.Type
		switch idType {
		case aadpodid.UserAssignedMSI:
			log.Infof("matched identityType:%v clientid:%s resource:%s", idType, utils.RedactClientID(clientID), rqResource)
			token, err := auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, rqResource)
			return token, clientID, err
		case aadpodid.ServicePrincipal:
			tenantid := v.Spec.TenantID
			log.Infof("matched identityType:%v tenantid:%s clientid:%s resource:%s", idType, tenantid, utils.RedactClientID(clientID), rqResource)
			secret, err := s.KubeClient.GetSecret(&v.Spec.ClientPassword)
			if err != nil {
				return nil, clientID, err
			}
			clientSecret := ""
			for _, v := range secret.Data {
				clientSecret = string(v)
				break
			}
			token, err := auth.GetServicePrincipalToken(tenantid, clientID, clientSecret, rqResource)
			return token, clientID, err
		default:
			return nil, clientID, fmt.Errorf("unsupported identity type %+v", idType)
		}
	}

	// We have not yet returned, so pass up an error
	return nil, "", fmt.Errorf("azureidentity is not configured for the pod")
}
