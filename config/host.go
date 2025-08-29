package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

	// identity files
	for _, identityFile := range identityFiles {
		if _, err := os.Stat(identityFile); err != nil {
			continue
		}
		host.IdentityFiles = append(host.IdentityFiles, identityFile)
	}

	host.FillAttrsWithSSHConfig()

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

func (host *Host) MatchTags(tags []string) bool {
	if len(tags) == 0 {
		return false
	}
	for _, tag := range tags {
		if slices.Contains(host.TagList, tag) {
			return true
		}
	}
	return false
}

func (host *Host) fillUsername() {
	if host.Username != "" {
		return
	}

	// fill username with patterns
	for _, pattern := range host.Patterns {
		if strings.ContainsAny(pattern, "*!?") {
			continue
		}
		host.Username = ssh_config.Get(pattern, "User")
		if host.Username != "" {
			break
		}
	}

	// fill username with host name
	if host.Username == "" {
		host.Username = ssh_config.Get(host.HostName, "User")
	}

	// fill username with environment variable
	if host.Username == "" {
		host.Username = os.Getenv("USER")
	}

	// default username
	if host.Username == "" {
		host.Username = "root"
	}
}

func (host *Host) fillPort() {
	if host.Port != 0 {
		return
	}

	// fill port with patterns
	for _, pattern := range host.Patterns {
		if strings.ContainsAny(pattern, "*!?") {
			continue
		}
		portInt, err := strconv.Atoi(ssh_config.Get(pattern, "Port"))
		if err != nil {
			continue
		}
		if portInt <= 0 || portInt >= 65536 {
			continue
		}
		host.Port = uint16(portInt)
		return
	}

	// fill port with host name
	if host.Port == 0 {
		portInt, err := strconv.Atoi(ssh_config.Get(host.HostName, "Port"))
		if err != nil {
			return
		}
		host.Port = uint16(portInt)
	}

	// fill port with host name
	if host.Port == 0 {
		host.Port = 22
	}
}

func (host *Host) fillProxyJump() {
	if host.ProxyJump != "" {
		return
	}

	// fill proxy jump with patterns
	for _, pattern := range host.Patterns {
		if strings.ContainsAny(pattern, "*!?") {
			continue
		}
		host.ProxyJump = ssh_config.Get(pattern, "ProxyJump")
	}

	// fill proxy jump with host name
	if host.ProxyJump == "" {
		host.ProxyJump = ssh_config.Get(host.HostName, "ProxyJump")
	}

	// jump list
	if host.ProxyJump != "" {
		for _, jump := range strings.Split(host.ProxyJump, ",") {
			if jump == "" {
				continue
			}
			jumpHost, err := NewHost("", jump, 0, "", host.IdentityFiles)
			if err != nil {
				continue
			}
			host.JumpList = append(host.JumpList, jumpHost)
		}
	}
}

func (host *Host) fillIdentityFiles() {
	if len(host.IdentityFiles) > 0 {
		return
	}

	// fill identity files with patterns
	for _, pattern := range host.Patterns {
		if strings.ContainsAny(pattern, "*!?") {
			continue
		}
		identityFiles := ssh_config.GetAll(pattern, "IdentityFile")
		for _, identityFile := range identityFiles {
			if _, err := os.Stat(identityFile); err == nil {
				host.IdentityFiles = append(host.IdentityFiles, identityFile)
			}
		}
	}

	// fill identity files with host name
	if len(host.IdentityFiles) == 0 {
		identityFiles := ssh_config.GetAll(host.HostName, "IdentityFile")
		for _, identityFile := range identityFiles {
			if _, err := os.Stat(identityFile); err == nil {
				host.IdentityFiles = append(host.IdentityFiles, identityFile)
			}
		}
	}

	// default identity file
	if len(host.IdentityFiles) == 0 {
		defaultIdentityFile := filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
		if _, err := os.Stat(defaultIdentityFile); err == nil {
			host.IdentityFiles = []string{defaultIdentityFile}
		}
	}
}

func (host *Host) FillAttrsWithSSHConfig() {
	host.fillUsername()
	host.fillPort()
	host.fillIdentityFiles()
	host.fillProxyJump() // must after identity files

	rawHostname := ssh_config.Get(host.HostName, "HostName")
	if rawHostname != "" {
		host.HostName = rawHostname
	}
}
