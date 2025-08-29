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

func hostFromSSHConfig(hostConfig *ssh_config.Host) (host *Host, err error) {
	host = &Host{Patterns: make([]string, 0)}

	// patterns
	for _, pattern := range hostConfig.Patterns {
		host.Patterns = append(host.Patterns, pattern.String())
	}

	// host info
	for _, node := range hostConfig.Nodes {
		switch node.(type) {
		case *ssh_config.Empty:
			continue
		case *ssh_config.KV:
			kv := node.(*ssh_config.KV)
			switch kv.Key {
			case "HostName":
				host.HostName = kv.Value
			case "User":
				host.Username = kv.Value
			case "Port":
				port, err := strconv.Atoi(kv.Value)
				if err != nil {
					return nil, err
				}
				if port <= 0 || port >= 65536 {
					return nil, fmt.Errorf("invalid port: %s", kv.Value)
				}
				host.Port = uint16(port)
			case "ProxyJump":
				host.ProxyJump = kv.Value
			case "IdentityFile":
				host.IdentityFiles = append(host.IdentityFiles, kv.Value)
			}
		}
	}

	return host, nil
}

func GetHostsFromSSHConfig() (hosts []*Host, err error) {
	hosts = make([]*Host, 0)

	configPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	if _, err := os.Stat(configPath); err != nil {
		return hosts, nil
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return hosts, err
	}

	sshConfig, err := ssh_config.DecodeBytes(configBytes)
	if err != nil {
		return hosts, err
	}

	for _, hostConfig := range sshConfig.Hosts {
		host, err := hostFromSSHConfig(hostConfig)
		if err != nil {
			return hosts, err
		}

		// 没有 HostName 的都是正则类型的配置
		if host.HostName == "" {
			continue
		}

		// 从正则配置中解析
		host.FillAttrsWithSSHConfig()

		// tags
		if hostConfig.EOLComment != "" {
			tagsRegex := regexp.MustCompile(`tags:([0-9a-zA-z_\-,]*)`)
			match := tagsRegex.FindStringSubmatch(hostConfig.EOLComment)
			if len(match) > 1 {
				for _, tag := range strings.Split(match[1], ",") {
					if tag == "" {
						continue
					}
					host.TagList = append(host.TagList, tag)
				}
			}
		}

		logger.Debugf("host: %+#v", host)
		hosts = append(hosts, host)
	}

	return hosts, nil
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
