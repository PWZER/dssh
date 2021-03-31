package utils

import (
	"net"
	"os"
	"strings"
)

func GetSSHConnectIP() string {
	// ssh login server
	sshConnInfo := strings.Split(os.Getenv("SSH_CONNECTION"), " ")
	if len(sshConnInfo) == 4 && len(sshConnInfo[2]) > 0 {
		return sshConnInfo[2]
	}
	return ""
}

func GetLocalIPs() (ips []string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && ipnet.IP.String() != "169.254.1.1" {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

func GetLocalIP() string {
	ip := GetSSHConnectIP()
	if ip != "" {
		return ip
	}

	ips := GetLocalIPs()
	if len(ips) > 0 {
		return ips[0]
	}
	return "0.0.0.0"
}
