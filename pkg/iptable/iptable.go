package iptable

import (
	"errors"

	"github.com/coreos/go-iptables/iptables"
)

// AddRule adds the required rule to the host's nat table.
func AddRule(sourcecidr, destIP, destPort, targetip, targetport string) error {
	if sourcecidr == "" {
		return errors.New("sourcecidr must be set")
	}
	if destIP == "" {
		return errors.New("destIP must be set")
	}
	if destPort == "" {
		return errors.New("destPort must be set")
	}
	if targetip == "" {
		return errors.New("targetip must be set")
	}
	if targetport == "" {
		return errors.New("targetport must be set")
	}

	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	return ipt.AppendUnique(
		"nat", "OUTPUT", "-p", "tcp", "-s", sourcecidr, "-d", destIP, "--dport", destPort,
		"-j", "DNAT", "--to-destination", targetip+":"+targetport,
	)
}
