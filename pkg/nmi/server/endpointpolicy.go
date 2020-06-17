// +build windows

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	client "github.com/Microsoft/hcnproxy/pkg/client"
	msg "github.com/Microsoft/hcnproxy/pkg/types"
	v1 "github.com/Microsoft/hcsshim"
	"k8s.io/klog"
)

const (
	InvalidOperation = "InvalidOperation"
	NotFound         = "NotFound"
)

type endpointPolicyError struct {
	errType string
	err     error
}

func (e *endpointPolicyError) Error() string {
	return fmt.Sprintf("%s: %v", e.errType, e.err)
}

// ApplyEndpointRoutePolicy applies the route policy against the pod ip endpoint
func ApplyEndpointRoutePolicy(podIP string, metadataIP string, metadataPort string, nmiIP string, nmiPort string) error {
	if podIP == "" {
		return errors.New("Missing IP Address")
	}

	endpoint, err := getEndpointByIP(podIP)

	if err != nil {
		if endpointPolicyError, ok := err.(*endpointPolicyError); ok {
			return fmt.Errorf("Get endpoint for Pod IP - %s. Error: %w", podIP, endpointPolicyError.err)
		}

		return fmt.Errorf("Get endpoint for Pod IP - %s. Error: %w", podIP, err)
	}

	err = addEndpointPolicy(endpoint, metadataIP, metadataPort, nmiIP, nmiPort)
	if err != nil {
		return fmt.Errorf("Could not add policy to endpoint - %s. Error: %w", endpoint.Id, err)
	}

	return nil
}

// DeleteEndpointRoutePolicy applies the route policy against the pod ip endpoint
func DeleteEndpointRoutePolicy(podIP string, metadataIP string) error {
	if podIP == "" {
		return errors.New("Missing IP Address")
	}

	endpoint, err := getEndpointByIP(podIP)

	if err != nil {
		if endpointPolicyError, ok := err.(*endpointPolicyError); ok {
			if endpointPolicyError.errType == InvalidOperation {
				return fmt.Errorf("Get endpoint for Pod IP - %s. Error: %w", podIP, endpointPolicyError.err)
			} else if endpointPolicyError.errType == NotFound {
				klog.Infof("No deleting action: no endpoint found for Pod IP - %s.", podIP)
				return nil
			}
		}
		return fmt.Errorf("Get endpoint for Pod IP - %s. Error: %w", podIP, err)
	}

	err = deleteEndpointPolicy(endpoint, metadataIP)
	if err != nil {
		return fmt.Errorf("Could't delete policy for endpoint - %s. Error: %v", endpoint.Id, err)
	}

	return nil
}

func getEndpointByIP(ip string) (*v1.HNSEndpoint, error) {
	klog.Infof("Getting endpoint for IP %s\n", ip)

	request := msg.HNSRequest{
		Entity:    msg.EndpointV1,
		Operation: msg.Enumerate,
		Request:   nil,
	}

	klog.Info("Enumerating all endpoints\n")
	response, err := callHcnProxyAgent(request)
	if err != nil {
		return nil, &endpointPolicyError{InvalidOperation, err}
	}

	var endpoints []v1.HNSEndpoint
	err = json.Unmarshal(response, &endpoints)
	if err != nil {
		return nil, &endpointPolicyError{InvalidOperation, err}
	}

	for _, ep := range endpoints {
		if ep.IPAddress.String() == ip {
			klog.Infof("Got endpoint for IP with id %s\n", ep.Id)
			return &ep, nil
		}
	}

	return nil, &endpointPolicyError{NotFound, fmt.Errorf("No endpoint found for Pod IP - %s.", ip)}
}

func addEndpointPolicy(endpoint *v1.HNSEndpoint, metadataIP string, metadataPort string, nmiIP string, nmiPort string) error {
	endpoint.Policies = updateEndpointPolicies(endpoint.Policies, metadataIP)

	klog.Infof("Trying to apply policy to endpoint %s\n", endpoint.Id)
	policy := &v1.ProxyPolicy{
		Type:        v1.Proxy,
		IP:          metadataIP,
		Port:        metadataPort,
		Destination: fmt.Sprintf("%s:%s", nmiIP, nmiPort),
	}

	jsonStr, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	endpoint.Policies = append(endpoint.Policies, jsonStr)

	jsonStr, err = json.Marshal(endpoint)
	if err != nil {
		return err
	}

	request := msg.HNSRequest{
		Entity:    msg.EndpointV1,
		Operation: msg.Modify,
		Request:   jsonStr,
	}

	klog.Infof("Adding policy to endpoint %s\n", endpoint.Id)
	_, err = callHcnProxyAgent(request)
	return err
}

func deleteEndpointPolicy(endpoint *v1.HNSEndpoint, metadataIP string) error {
	endpoint.Policies = updateEndpointPolicies(endpoint.Policies, metadataIP)

	jsonStr, err := json.Marshal(endpoint)
	if err != nil {
		return err
	}

	request := msg.HNSRequest{
		Entity:    msg.EndpointV1,
		Operation: msg.Modify,
		Request:   jsonStr,
	}

	klog.Infof("Deleting policy from endpoint %s\n", endpoint.Id)
	_, err = callHcnProxyAgent(request)

	return err
}

func callHcnProxyAgent(req msg.HNSRequest) ([]byte, error) {
	retryCount := 1
	maxRetryCount := 4
	var sleepFactor time.Duration = 1

	klog.Info("Calling HNS Agent")

	for {
		response, err := callHcnProxyAgentInternal(req)
		if err != nil {
			if retryCount > maxRetryCount {
				klog.Info("Calling HNS Agent failed after all retries, giving up")
				return nil, err
			}

			klog.Infof("Calling HNS Agent failed, will retry in %s, Error: %w", sleepFactor, err)
			time.Sleep(sleepFactor * time.Second)
			sleepFactor = sleepFactor * 2
			retryCount++
			continue
		}

		klog.Info("Call to HNS Agent successful")
		return response, nil
	}
}

func callHcnProxyAgentInternal(req msg.HNSRequest) ([]byte, error) {
	res := client.InvokeHNSRequest(req)
	if res.Error != nil {
		return nil, res.Error
	}

	b, _ := json.Marshal(res)
	klog.Infof("Server response: %s", string(b))

	return res.Response, nil
}

func updateEndpointPolicies(policies []json.RawMessage, metadataIP string) []json.RawMessage {
	count := -1
	index := 0
	var proxyPolicy v1.ProxyPolicy

	endpointPolicies := policies

	for i, p := range policies {
		err := json.Unmarshal(p, &proxyPolicy)
		if err == nil && proxyPolicy.IP == metadataIP {
			count++
			index = i - count
			endpointPolicies = append(endpointPolicies[:index], endpointPolicies[index+1:]...)
		}
	}

	return endpointPolicies
}
