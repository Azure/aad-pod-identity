package iptables

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
)

var (
	tablename       = "nat"
	customchainname = "aad-metadata"
)

// AddCustomChain adds the rule to the host's nat table custom chain
func AddCustomChain(sourcecidr, destIP, destPort, targetip, targetport string) error {
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
	if err := ensureCustomChain(ipt, sourcecidr, destIP, destPort, targetip, targetport); err != nil {
		return err
	}
	if err := placeCustomChainInChain(ipt, tablename, "OUTPUT"); err != nil {
		return err
	}
	if err := placeCustomChainInChain(ipt, tablename, "PREROUTING"); err != nil {
		return err
	}

	return nil
}

// LogCustomChain logs added rules to the custom chain
func LogCustomChain() error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	rules, err := ipt.List(tablename, customchainname)
	if err != nil {
		return err
	}

	log.Infof("Rules for table(%s) chain(%s) rules(%+v)", tablename, customchainname, strings.Join(rules, ", "))

	return nil
}

func generateRulesSpec(sourcecidr, destIP, destPort, targetip, targetport string) *[]string {
	rules := []string{
		// 1. metadata endpoint to nmi rule
		fmt.Sprintf("-p tcp -s %s -d %s --dport %s -j DNAT --to-destination %s:%s",
			sourcecidr, destIP, destPort, targetip, targetport),
		// 2. jump rule
		"-j RETURN",
	}

	return &rules
}

//	iptables -t nat -I "chain" 1 -j "customchainname"
func placeCustomChainInChain(ipt *iptables.IPTables, table, chain string) error {
	exists, err := ipt.Exists(table, chain, "-j", customchainname)
	if err != nil || !exists {
		if err := ipt.Insert(table, chain, 1, "-j", customchainname); err != nil {
			return err
		}
	}

	return nil
}

func ensureCustomChain(ipt *iptables.IPTables, sourcecidr, destIP, destPort,
	targetip, targetport string) error {
	rules, err := ipt.List(tablename, customchainname)
	if err != nil {
		err = ipt.NewChain(tablename, customchainname)
		if err != nil {
			return err
		}
	}

	if len(rules) == 2 {
		return nil
	}

	if err := flushCreateCustomChainrules(ipt, sourcecidr, destIP, destPort,
		targetip, targetport); err != nil {
		return err
	}

	return nil
}

func flushCreateCustomChainrules(ipt *iptables.IPTables, sourcecidr, destIP, destPort, targetip, targetport string) error {
	if err := ipt.ClearChain(tablename, customchainname); err != nil {
		return err
	}

	if err := ipt.AppendUnique(
		tablename, customchainname, "-p", "tcp", "-s", sourcecidr, "-d", destIP, "--dport", destPort,
		"-j", "DNAT", "--to-destination", targetip+":"+targetport); err != nil {
		return err
	}

	if err := ipt.AppendUnique(
		tablename, customchainname, "-j", "RETURN"); err != nil {
		return err
	}

	return nil
}
