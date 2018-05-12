package iptable

import (
	"errors"

	"github.com/coreos/go-iptables/iptables"
)

// AddRule adds the required rule to the host's nat table.
func AddRule(podcidr, metadataAddress, nodeip, nmiport string) error {
	if podcidr == "" {
		return errors.New("podcidr must be set")
	}

	if nodeip == "" {
		return errors.New("nodeip must be set")
	}

	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	return ipt.AppendUnique(
		"nat", "PREROUTING", "-p", "tcp", "-s", podcidr, "-d", metadataAddress, "--dport", "80",
		"-j", "DNAT", "--to-destination", nodeip+":"+nmiport,
	)
}
