package server

import (
	"context"
	"fmt"
	"strings"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity"
	auth "github.com/Azure/aad-pod-identity/pkg/auth"
	k8s "github.com/Azure/aad-pod-identity/pkg/k8s"
	"github.com/Azure/aad-pod-identity/pkg/nmi"
	utils "github.com/Azure/aad-pod-identity/pkg/utils"
	"github.com/Azure/go-autorest/autorest/adal"
	"k8s.io/klog"
)

// ManagedClient implements the TokenClient interface
type ManagedClient struct {
	nmi.TokenClient
	KubeClient   k8s.Client
	IsNamespaced bool
}

// NewManagedTokenClient creates new managed token client
func NewManagedTokenClient(client k8s.Client, isNamespaced bool) *ManagedClient {
	return &ManagedClient{
		KubeClient:   client,
		IsNamespaced: isNamespaced,
	}
}

// GetIdentities gets the azure identity that matches the podns/podname and client id
func (mc *ManagedClient) GetIdentities(ctx context.Context, podns, podname, clientID string) (*aadpodid.AzureIdentity, error) {
	// get pod object to retrieve labels
	pod, err := mc.KubeClient.GetPod(podns, podname)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s, err: %+v", podns, podname, err)
	}
	// get all the azure identities based on azure identity bindings
	azureIdentities, err := mc.KubeClient.ListPodIdsWithBinding(podns, pod.Labels)

	for _, id := range azureIdentities {
		// if client id exists in the request, then send the first identity that matched the client id
		if len(clientID) != 0 && id.Spec.ClientID == clientID {
			return &id, nil
		}
		// if client doesn't exist in the request, then return the first identity in the same namespace as the pod
		if len(clientID) == 0 && strings.EqualFold(id.Namespace, podns) {
			klog.Infof("No clientID in request. %s/%s has been matched with azure identity %s/%s", podns, podname, id.Namespace, id.Name)
			return &id, nil
		}
	}
	return nil, fmt.Errorf("no matching azure identity found for pod")
}

// GetToken ...
func (mc *ManagedClient) GetToken(ctx context.Context, rqClientID, rqResource string, podID aadpodid.AzureIdentity) (token *adal.Token, err error) {
	rqHasClientID := len(rqClientID) != 0
	clientID := podID.Spec.ClientID
	if rqHasClientID && !strings.EqualFold(rqClientID, clientID) {
		klog.Warningf("clientid mismatch, requested:%s available:%s", rqClientID, clientID)
	}

	idType := podID.Spec.Type
	switch idType {
	case aadpodid.UserAssignedMSI:
		klog.Infof("matched identityType:%v clientid:%s resource:%s", idType, utils.RedactClientID(clientID), rqResource)
		token, err := auth.GetServicePrincipalTokenFromMSIWithUserAssignedID(clientID, rqResource)
		return token, err
	case aadpodid.ServicePrincipal:
		tenantid := podID.Spec.TenantID
		klog.Infof("matched identityType:%v tenantid:%s clientid:%s resource:%s", idType, tenantid, utils.RedactClientID(clientID), rqResource)
		secret, err := mc.KubeClient.GetSecret(&podID.Spec.ClientPassword)
		if err != nil {
			return nil, err
		}
		clientSecret := ""
		for _, v := range secret.Data {
			clientSecret = string(v)
			break
		}
		token, err := auth.GetServicePrincipalToken(tenantid, clientID, clientSecret, rqResource)
		return token, err
	default:
		return nil, fmt.Errorf("unsupported identity type %+v", idType)
	}
}
