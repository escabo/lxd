package lxd

import (
	"fmt"
	"net"
	"net/http"

	"github.com/canonical/lxd/shared/api"
)

// GetNetworkLoadBalancerAddresses returns a list of network load balancer listen addresses.
func (r *ProtocolLXD) GetNetworkLoadBalancerAddresses(networkName string) ([]string, error) {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return nil, err
	}

	// Fetch the raw URL values.
	urls := []string{}
	u := api.NewURL().Path("networks", networkName, "load-balancers")
	_, err = r.queryStruct(http.MethodGet, u.String(), nil, "", &urls)
	if err != nil {
		return nil, err
	}

	// Parse it.
	return urlsToResourceNames(u.String(), urls...)
}

// GetNetworkLoadBalancers returns a list of Network load balancer structs.
func (r *ProtocolLXD) GetNetworkLoadBalancers(networkName string) ([]api.NetworkLoadBalancer, error) {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return nil, err
	}

	loadBalancers := []api.NetworkLoadBalancer{}

	// Fetch the raw value.
	u := api.NewURL().Path("networks", networkName, "load-balancers").WithQuery("recursion", "1")
	_, err = r.queryStruct(http.MethodGet, u.String(), nil, "", &loadBalancers)
	if err != nil {
		return nil, err
	}

	return loadBalancers, nil
}

// GetNetworkLoadBalancer returns a Network load balancer entry for the provided network and listen address.
func (r *ProtocolLXD) GetNetworkLoadBalancer(networkName string, listenAddress string) (*api.NetworkLoadBalancer, string, error) {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return nil, "", err
	}

	loadBalancer := api.NetworkLoadBalancer{}

	// Fetch the raw value.
	u := api.NewURL().Path("networks", networkName, "load-balancers", listenAddress)
	etag, err := r.queryStruct(http.MethodGet, u.String(), nil, "", &loadBalancer)
	if err != nil {
		return nil, "", err
	}

	return &loadBalancer, etag, nil
}

// CreateNetworkLoadBalancer defines a new network load balancer using the provided struct.
func (r *ProtocolLXD) CreateNetworkLoadBalancer(networkName string, loadBalancer api.NetworkLoadBalancersPost) error {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return err
	}

	listenAddressIP := net.ParseIP(loadBalancer.ListenAddress)
	if listenAddressIP == nil {
		return fmt.Errorf("Invalid network load balancer listen address: %s", loadBalancer.ListenAddress)
	}

	if listenAddressIP.IsUnspecified() {
		err := r.CheckExtension("network_allocate_external_ips")
		if err != nil {
			return err
		}
	}

	// Send the request.
	u := api.NewURL().Path("networks", networkName, "load-balancers")
	_, _, err = r.query(http.MethodPost, u.String(), loadBalancer, "")
	if err != nil {
		return err
	}

	return nil
}

// UpdateNetworkLoadBalancer updates the network load balancer to match the provided struct.
func (r *ProtocolLXD) UpdateNetworkLoadBalancer(networkName string, listenAddress string, loadBalancer api.NetworkLoadBalancerPut, ETag string) error {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return err
	}

	// Send the request.
	u := api.NewURL().Path("networks", networkName, "load-balancers", listenAddress)
	_, _, err = r.query(http.MethodPut, u.String(), loadBalancer, ETag)
	if err != nil {
		return err
	}

	return nil
}

// DeleteNetworkLoadBalancer deletes an existing network load balancer.
func (r *ProtocolLXD) DeleteNetworkLoadBalancer(networkName string, listenAddress string) error {
	err := r.CheckExtension("network_load_balancer")
	if err != nil {
		return err
	}

	// Send the request.
	u := api.NewURL().Path("networks", networkName, "load-balancers", listenAddress)
	_, _, err = r.query(http.MethodDelete, u.String(), nil, "")
	if err != nil {
		return err
	}

	return nil
}
