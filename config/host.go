package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/PWZER/dssh/logger"
	"github.com/kevinburke/ssh_config"
)

type Host struct {
	Patterns      []string
	HostName      string
	Username      string
	Port          uint16
	ProxyJump     string
	TagList       []string
	JumpList      []*Host
	IdentityFiles []string
}

func NewHost(username, hostname string, port uint16, proxyJump string, identityFiles []string) (host *Host, err error) {
	host = &Host{
		Username:      username,
		HostName:      hostname,
		Port:          port,
		ProxyJump:     proxyJump,
		IdentityFiles: []string{},
	}

	// hostname
	if host.HostName == "" {
		return nil, fmt.Errorf("hostname is required non-empty string!")
	}

	// parse format user@hostname
	if strings.Contains(host.HostName, "@") {
		if host.Username != "" {
			return nil, fmt.Errorf("username is already set: %v", host.HostName)
		}

		parts := strings.Split(host.HostName, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid hostname format: %v", host.HostName)
		}
		host.Username = parts[0]
		host.HostName = parts[1]
	}

	// parse format hostname:port
	if strings.Contains(host.HostName, ":") {
		if host.Port != 0 {
			return nil, fmt.Errorf("port is already set: %v", host.HostName)
		}

		parts := strings.Split(host.HostName, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid hostname format: %v", host.HostName)
		}

		if portInt, err := strconv.Atoi(parts[1]); err != nil {
			return nil, fmt.Errorf("invalid port format: %v", parts[1])
		} else if portInt <= 0 || portInt >= 65536 {
			return nil, fmt.Errorf("invalid port format: %v", parts[1])
		} else {
			host.Port = uint16(portInt)
		}
		host.HostName = parts[0]
	}

	// port
	if host.Port == 0 {
		portInt, err := strconv.Atoi(ssh_config.Get(host.HostName, "Port"))
		if err != nil {
			return nil, fmt.Errorf("invalid port format: %v", host.HostName)
		} else if portInt <= 0 || portInt >= 65536 {
			return nil, fmt.Errorf("invalid port format: %v", host.HostName)
		} else {
			host.Port = uint16(portInt)
		}

		// default port
		if host.Port == 0 {
			host.Port = 22
		}
	}

	// username
	if host.Username == "" {
		host.Username = ssh_config.Get(host.HostName, "User")
		if host.Username == "" {
			host.Username = os.Getenv("USER")
		}
		if host.Username == "" {
			host.Username = "root"
		}
	}

	// identity files
	for _, identityFile := range identityFiles {
		if _, err := os.Stat(identityFile); err != nil {
			continue
		}
		host.IdentityFiles = append(host.IdentityFiles, identityFile)
	}

	if len(host.IdentityFiles) == 0 {
		identityFiles := ssh_config.GetAll(host.HostName, "IdentityFile")
		for _, identityFile := range identityFiles {
			if _, err := os.Stat(identityFile); err != nil {
				continue
			}
			host.IdentityFiles = append(host.IdentityFiles, identityFile)
		}

		// default identity file
		if len(host.IdentityFiles) == 0 {
			host.IdentityFiles = []string{filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")}
		}
	}

	// proxy jump
	if host.ProxyJump == "" {
		host.ProxyJump = ssh_config.Get(host.HostName, "ProxyJump")
	}

	// jump list
	for _, jump := range strings.Split(host.ProxyJump, ",") {
		if jump == "" {
			continue
		}
		jumpHost, err := NewHost("", jump, 0, "", host.IdentityFiles)
		if err != nil {
			return nil, err
		}
		host.JumpList = append(host.JumpList, jumpHost)
	}

	rawHostname := ssh_config.Get(host.HostName, "HostName")
	if rawHostname != "" {
		host.HostName = rawHostname
	}

	logger.Debugf("host: %+#v", host)
	return host, nil
}

func (host *Host) EndPoint() string {
	if host.Port == 0 {
		return host.HostName
	}
	return fmt.Sprintf("%v:%v", host.HostName, host.Port)
}

func (host *Host) Summary() string {
	if host.Username == "" {
		return host.EndPoint()
	}
	return fmt.Sprintf("%v@%v", host.Username, host.EndPoint())
}

func (host *Host) JumpString() string {
	hosts := make([]string, 0)
	for _, host := range host.JumpList {
		hosts = append(hosts, host.Summary())
	}
	return strings.Join(hosts, ",")
}

func CheckTags(tags string) error {
	if matched, err := regexp.MatchString(`[0-9a-zA-z_\-,]*`, tags); err != nil {
		return err
	} else if !matched {
		return fmt.Errorf("Invalid tags!")
	}
	return nil
}

func GetHostNames() (names []string) {
	hosts, err := GetHostsFromSSHConfig()
	if err != nil {
		return nil
	}
	for _, host := range hosts {
		for _, pattern := range host.Patterns {
			if strings.ContainsAny(pattern, "*!?") {
				continue
			}
			names = append(names, pattern)
		}
	}
	return names
}
