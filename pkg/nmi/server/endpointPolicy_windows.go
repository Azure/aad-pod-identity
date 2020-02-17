package server

import (
	"errors"
	"fmt"

	"github.com/Microsoft/hcnproxy"
	v1 "github.com/Microsoft/hcsshim"
	"k8s.io/klog"
)

// ApplyEndpointRoutePolicy applies the route policy against the pod ip endpoint
func ApplyEndpointRoutePolicy(podIP string, metadataIP string, metadataPort string, nmiIP string, nmiPort string) error {
	if len(ip) <= 0 {
		return errors.New("Missing IP Address")
	}

	endpoint, err := getEndpointByIP(ip)
	if err != nil {
		return err
	}

	er := addEndpointPolicy(endpoint, metadataIP, metadataPort, nmiIP, nmiPort)
	if er != nil {
		return err
	}

	return nil
}

// DeleteEndpointRoutePolicy applies the route policy against the pod ip endpoint
func DeleteEndpointRoutePolicy(podIP string) error {
	if len(ip) <= 0 {
		return errors.New("Missing IP Address")
	}

	endpoint, err := getEndpointByIP(ip)
	if err != nil {
		return err
	}

	er := deleteEndpointPolicy(endpoint)
	if er != nil {
		return err
	}
}

func getEndpointByIP(ip string) (*v1.HNSEndpoint, error) {
	fmt.Printf("Getting endpoint for IP %s\n", ip)

	request := HNSRequest{
		Entity:    EndpointV1,
		Operation: Enumerate,
		Request:   nil,
	}

	klog.Info("Enumerating all endpoints")
	response, e := callHcnProxyAgent(request)
	if e != nil {
		return nil, e
	}

	var endpoints []v1.HNSEndpoint
	err := json.Unmarshal(response, &endpoints)
	if err != nil {
		return nil, err
	}

	for _, j := range endpoints {
		if j.IPAddress.String() == ip {
			klog.Info("Got endpoint for IP with id %s\n", j.Id)
			return &j, nil
		}
	}

	return nil, fmt.Errorf("no endpoint found for IP address - %s", ip)
}

func addEndpointPolicy(endpoint *v1.HNSEndpoint, metadataIP string, metadataPort, string, nmiIP string, nmiPort string) error {

	if checkProxyPolicyExists(endpoint) == true {
		klog.Info("Proxy policy exists for endpoint %s. Skipping...\n", endpoint.Id)
		return nil
	}

	klog.Info("No proxy policy exists for the endpoint. Trying to apply policy to endpoint %s\n", endpoint.Id)
	pp := &v1.ProxyPolicy{
		Type:        v1.Proxy,
		IP:          metadataIP,
		Port:        metadataPort,
		Destination: fmt.SPrintf("%s:%s", nmiIP, nmiPort),
	}

	jsonStr, err := json.Marshal(pp)
	if err != nil {
		return err
	}
	endpoint.Policies = append(endpoint.Policies, jsonStr)

	jsonStr, e := json.Marshal(endpoint)
	if e != nil {
		return err
	}

	request := HNSRequest{
		Entity:    EndpointV1,
		Operation: Modify,
		Request:   jsonStr,
	}

	klog.Info("Adding policy to endpoint %s\n", endpoint.Id)
	_, er := callHcnProxyAgent(request)
	return er
}

func deleteEndpointPolicy(endpoint *v1.HNSEndpoint) error {
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

	jsonStr, e := json.Marshal(endpoint)
	if e != nil {
		return e
	}

	request := HNSRequest{
		Entity:    EndpointV1,
		Operation: Modify,
		Request:   jsonStr,
	}

	klog.Info("Deleting policy from endpoint %s\n", endpoint.Id)
	_, er := callHcnProxyAgent(request)

	return er
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

func callHcnProxyAgent(req HNSRequest) ([]byte, error) {
	klog.Info("Calling HNS Agent")
	res := InvokeHNSRequest(req)
	if res.Error != nil {
		return nil, res.Error
	}

	return res.Response, nil
}
