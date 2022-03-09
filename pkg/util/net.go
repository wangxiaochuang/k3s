package util

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/urfave/cli"
	apinet "k8s.io/apimachinery/pkg/util/net"
)

func JoinIPs(elems []net.IP) string {
	var strs []string
	for _, elem := range elems {
		strs = append(strs, elem.String())
	}
	return strings.Join(strs, ",")
}

func JoinIPNets(elems []*net.IPNet) string {
	var strs []string
	for _, elem := range elems {
		strs = append(strs, elem.String())
	}
	return strings.Join(strs, ",")
}

func GetFirst4Net(elems []*net.IPNet) (*net.IPNet, error) {
	for _, elem := range elems {
		if elem == nil || elem.IP.To4() == nil {
			continue
		}
		return elem, nil
	}
	return nil, errors.New("no IPv4 CIDRs found")
}

func GetFirst4(elems []net.IP) (net.IP, error) {
	for _, elem := range elems {
		if elem == nil || elem.To4() == nil {
			continue
		}
		return elem, nil
	}
	return nil, errors.New("no IPv4 address found")
}

func GetFirst4String(elems []string) (string, error) {
	ips := []net.IP{}
	for _, elem := range elems {
		for _, v := range strings.Split(elem, ",") {
			ips = append(ips, net.ParseIP(v))
		}
	}
	ip, err := GetFirst4(ips)
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

func JoinIP4Nets(elems []*net.IPNet) string {
	var strs []string
	for _, elem := range elems {
		if elem != nil && elem.IP.To4() != nil {
			strs = append(strs, elem.String())
		}
	}
	return strings.Join(strs, ",")
}

func JoinIP6Nets(elems []*net.IPNet) string {
	var strs []string
	for _, elem := range elems {
		if elem != nil && elem.IP.To4() == nil {
			strs = append(strs, elem.String())
		}
	}
	return strings.Join(strs, ",")
}

func GetHostnameAndIPs(name string, nodeIPs cli.StringSlice) (string, []net.IP, error) {
	ips := []net.IP{}
	if len(nodeIPs) == 0 {
		hostIP, err := apinet.ChooseHostInterface()
		if err != nil {
			return "", nil, err
		}
		ips = append(ips, hostIP)
	} else {
		var err error
		ips, err = ParseStringSliceToIPs(nodeIPs)
		if err != nil {
			return "", nil, fmt.Errorf("invalid node-ip: %w", err)
		}
	}

	if name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return "", nil, err
		}
		name = hostname
	}

	name = strings.ToLower(name)

	return name, ips, nil
}

func ParseStringSliceToIPs(s cli.StringSlice) ([]net.IP, error) {
	var ips []net.IP
	for _, unparsedIP := range s {
		for _, v := range strings.Split(unparsedIP, ",") {
			ip := net.ParseIP(v)
			if ip == nil {
				return nil, fmt.Errorf("invalid ip format '%s'", v)
			}
			ips = append(ips, ip)
		}
	}

	return ips, nil
}
