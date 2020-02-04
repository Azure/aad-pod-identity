// +build windows

package server

import (
	"encoding/json"
	"errors"
	"fmt"

	client "github.com/Microsoft/hcnproxy/pkg/client"
	msg "github.com/Microsoft/hcnproxy/pkg/types"
	v1 "github.com/Microsoft/hcsshim"
	"k8s.io/klog"
)

// ApplyEndpointRoutePolicy applies the route policy against the pod ip endpoint
func ApplyEndpointRoutePolicy(podIP string, metadataIP string, metadataPort string, nmiIP string, nmiPort string) error {
	if podIP == "" {
		return errors.New("Missing IP Address")
	}

	endpoint, err := getEndpointByIP(podIP)
	if err != nil {
		return fmt.Errorf("No endpoint found for Pod IP - %s. Error: %w", podIP, err)
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
		return fmt.Errorf("No endpoint found for Pod IP - %s. Error: %w", podIP, err)
	}

	err = deleteEndpointPolicy(endpoint, metadataIP)
	if err != nil {
		return fmt.Errorf("Could't delete policy for endpoint - %s. Error: %w", endpoint.Id, err)
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
		return nil, err
	}

	var endpoints []v1.HNSEndpoint
	err = json.Unmarshal(response, &endpoints)
	if err != nil {
		return nil, err
	}

	for _, ep := range endpoints {
		if ep.IPAddress.String() == ip {
			klog.Infof("Got endpoint for IP with id %s\n", ep.Id)
			return &ep, nil
		}
	}

	return nil, fmt.Errorf("No endpoint found for IP address - %s", ip)
}

func addEndpointPolicy(endpoint *v1.HNSEndpoint, metadataIP string, metadataPort string, nmiIP string, nmiPort string) error {

	if checkProxyPolicyExists(endpoint) == true {
		klog.Infof("Proxy policy exists for endpoint %s. Skipping...\n", endpoint.Id)
		return nil
	}

	klog.Infof("No proxy policy exists for the endpoint. Trying to apply policy to endpoint %s\n", endpoint.Id)
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
	index := 0
	var proxyPolicy v1.ProxyPolicy
	for i, p := range endpoint.Policies {
		err := json.Unmarshal(p, &proxyPolicy)
		if err != nil && proxyPolicy.IP == metadataIP {
			index = i
			break
		}
	}

	endpoint.Policies = append(endpoint.Policies[:index], endpoint.Policies[index+1:]...)

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

func checkProxyPolicyExists(endpoint *v1.HNSEndpoint) bool {
	var proxyPolicy v1.ProxyPolicy
	for _, p := range endpoint.Policies {
		err := json.Unmarshal(p, &proxyPolicy)
		if err != nil {
			return true
		}
	}

	return false
}

func callHcnProxyAgent(req msg.HNSRequest) ([]byte, error) {
	klog.Info("Calling HNS Agent")
	res := client.InvokeHNSRequest(req)
	if res.Error != nil {
		return nil, res.Error
	}

	b, _ := json.Marshal(res)
	klog.Infof("Server response: %s", string(b))

	return res.Response, nil
}
