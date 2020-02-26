package nmi

import (
	"context"
	"fmt"
	"strings"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	utils "github.com/Azure/aad-pod-identity/pkg/utils"
	"github.com/Azure/go-autorest/autorest/adal"

	"k8s.io/klog"
)

// StandardClient implements the TokenClient interface
type StandardClient struct {
	TokenClient
	KubeClient                         k8s.Client
	ListPodIDsRetryAttemptsForCreated  int
	ListPodIDsRetryAttemptsForAssigned int
	ListPodIDsRetryIntervalInSeconds   int
	IsNamespaced                       bool
}

// NewStandardTokenClient creates new standard nmi client
func NewStandardTokenClient(client k8s.Client, config Config) (*StandardClient, error) {
	return &StandardClient{
		KubeClient:                         client,
		ListPodIDsRetryAttemptsForCreated:  config.RetryAttemptsForCreated,
		ListPodIDsRetryAttemptsForAssigned: config.RetryAttemptsForAssigned,
		ListPodIDsRetryIntervalInSeconds:   config.FindIdentityRetryIntervalInSeconds,
		IsNamespaced:                       config.Namespaced,
	}, nil
}

// GetIdentities ...
func (sc *StandardClient) GetIdentities(ctx context.Context, podns, podname, clientID string) (*aadpodid.AzureIdentity, error) {
	podIDs, identityInCreatedStateFound, err := sc.listPodIDsWithRetry(ctx, podns, podname, clientID)
	if err != nil {
		// if identity not found in created state return nil identity which is then used to send 403 error
		if !identityInCreatedStateFound {
			return nil, err
		}
		// identity found in created state but there was an error, then return empty struct which will result in 404 error
		return &aadpodid.AzureIdentity{}, err
	}

	// filter out if we are in namespaced mode
	filterPodIdentities := []aadpodid.AzureIdentity{}
	for _, val := range podIDs {
		if sc.IsNamespaced || aadpodid.IsNamespacedIdentity(&val) {
			// namespaced mode
			if val.Namespace == podns {
				// matched namespace
				filterPodIdentities = append(filterPodIdentities, val)
			} else {
				// unmatched namespaced
				klog.Errorf("pod:%s/%s has identity %s/%s but identity is namespaced will be ignored", podns, podname, val.Name, val.Namespace)
			}
		} else {
			// not in namespaced mode
			filterPodIdentities = append(filterPodIdentities, val)
		}
	}

	for _, id := range filterPodIdentities {
		// if client doesn't exist in the request, then return the first identity
		if len(clientID) == 0 {
			klog.Infof("No clientID in request. %s/%s has been matched with azure identity %s/%s", podns, podname, id.Namespace, id.Name)
			return &id, nil
		}
		// if client id exists in the request, then send the first identity that matched the client id
		if len(clientID) != 0 && id.Spec.ClientID == clientID {
			return &id, nil
		}
	}

	return nil, fmt.Errorf("azureidentity is not configured for the pod")
}

// listPodIDsWithRetry returns a list of matched identities in Assigned state, boolean indicating if at least an identity was found in Created state and error if any
func (sc *StandardClient) listPodIDsWithRetry(ctx context.Context, podns, podname, rqClientID string) ([]aadpodid.AzureIdentity, bool, error) {
	attempt := 0
	var err error
	var idStateMap map[string][]aadpodid.AzureIdentity

	// this loop will run to ensure we have assigned identities before we return. If there are no assigned identities in created state within 80s (16 retries * 5s wait) then we return an error.
	// If we get an assigned identity in created state within 80s, then loop will continue until 100s to find assigned identity in assigned state.
	// Retry interval for CREATED state is set to 80s because avg time for identity to be assigned to the node is 35-37s.
	for attempt < sc.ListPodIDsRetryAttemptsForCreated+sc.ListPodIDsRetryAttemptsForAssigned {
		idStateMap, err = sc.KubeClient.ListPodIds(podns, podname)
		if err == nil {
			if len(rqClientID) == 0 {
				// check to ensure backward compatability with assignedIDs that have no state
				// assigned identites created with old version of mic will not contain a state. So first we check to see if an assigned identity with
				// no state exists that matches req client id.
				if len(idStateMap[""]) != 0 {
					klog.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
					return idStateMap[""], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDAssigned]) != 0 {
					return idStateMap[aadpodid.AssignedIDAssigned], true, nil
				}
				if len(idStateMap[aadpodid.AssignedIDCreated]) == 0 && attempt >= sc.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, sc.ListPodIDsRetryAttemptsForCreated, sc.ListPodIDsRetryIntervalInSeconds, err)
				}
			} else {
				// if client id exists in request, we need to ensure the identity with this client
				// exists and is in Assigned state
				// check to ensure backward compatability with assignedIDs that have no state
				for _, podID := range idStateMap[""] {
					if strings.EqualFold(rqClientID, podID.Spec.ClientID) {
						klog.Warningf("found assignedIDs with no state for pod:%s/%s. AssignedIDs created with old version of mic.", podns, podname)
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
				if !foundMatch && attempt >= sc.ListPodIDsRetryAttemptsForCreated {
					return nil, false, fmt.Errorf("getting assigned identities for pod %s/%s in CREATED state failed after %d attempts, retry duration [%d]s. Error: %v",
						podns, podname, sc.ListPodIDsRetryAttemptsForCreated, sc.ListPodIDsRetryIntervalInSeconds, err)
				}
			}
		}
		attempt++

		select {
		case <-time.After(time.Duration(sc.ListPodIDsRetryIntervalInSeconds) * time.Second):
		case <-ctx.Done():
			err = ctx.Err()
			return nil, true, err
		}
		klog.V(4).Infof("failed to get assigned ids for pod:%s/%s in ASSIGNED state, retrying attempt: %d", podns, podname, attempt)
	}
	return nil, true, fmt.Errorf("getting assigned identities for pod %s/%s in ASSIGNED state failed after %d attempts, retry duration [%d]s. Error: %v",
		podns, podname, sc.ListPodIDsRetryAttemptsForCreated+sc.ListPodIDsRetryAttemptsForAssigned, sc.ListPodIDsRetryIntervalInSeconds, err)
}

// GetToken ...
func (sc *StandardClient) GetToken(ctx context.Context, rqClientID, rqResource string, azureID aadpodid.AzureIdentity) (token *adal.Token, err error) {
	rqHasClientID := len(rqClientID) != 0
	clientID := azureID.Spec.ClientID

	idType := azureID.Spec.Type
	switch idType {
	case aadpodid.UserAssignedMSI:
		if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
			klog.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
		}
		klog.Infof("matched identityType:%v clientid:%s resource:%s", idType, utils.RedactClientID(clientID), rqResource)
		token, err := auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, rqResource)
		return token, err
	case aadpodid.ServicePrincipal:
		tenantID := azureID.Spec.TenantID
		klog.Infof("matched identityType:%v tenantid:%s clientid:%s resource:%s", idType, tenantID, utils.RedactClientID(clientID), rqResource)
		secret, err := sc.KubeClient.GetSecret(&azureID.Spec.ClientPassword)
		if err != nil {
			return nil, err
		}
		clientSecret := ""
		for _, v := range secret.Data {
			clientSecret = string(v)
			break
		}
		token, err := auth.GetServicePrincipalToken(tenantID, clientID, clientSecret, rqResource)
		return token, err
	case aadpodid.ServicePrincipalCertificate:
		tenantID := azureID.Spec.TenantID
		klog.Infof("matched identityType:%v tenantid:%s clientid:%s resource:%s", idType, tenantID, utils.RedactClientID(clientID), rqResource)
		secret, err := sc.KubeClient.GetSecret(&azureID.Spec.ClientPassword)
		if err != nil {
			return nil, err
		}
		certificate, password := secret.Data["certificate"], secret.Data["password"]
		token, err := auth.GetServicePrincipalTokenWithCertificate(tenantID, clientID,
			certificate, string(password), rqResource)
		return token, err
	default:
		return nil, fmt.Errorf("unsupported identity type %+v", idType)
	}
}
