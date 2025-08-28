package config

import (
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/PWZER/dssh/logger"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

type ConfigType struct {
	ModulesDir  string `yaml:"modulesDir,omitempty"`
	SSHAuthSock string `yaml:"sshAuthSock,omitempty"`
}

var Config = &ConfigType{}

func getSSHAuthSock() (sock string) {
	sock = os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		homeDir, _ := homedir.Dir()
		sock = path.Join(homeDir, ".ssh/ssh_auth_sock")
	}
	return sock
}

func LoadConfig() error {
	if err := viper.Unmarshal(Config); err != nil {
		return err
	}

	if Config.SSHAuthSock == "" {
		Config.SSHAuthSock = getSSHAuthSock()
	}
	return nil
}

func FilteredHosts(name string, user string, tags string) (hosts []*Host, err error) {
	hosts, err = GetHostsFromSSHConfig()
	if err != nil {
		return nil, err
	}

	if name != "" {
		newHosts := make([]*Host, 0)
		for _, host := range hosts {
			if host.HostName == name || strings.Contains(host.HostName, name) {
				newHosts = append(newHosts, host)
			}
		}
		logger.Debugf("Filtered hosts by name: %+#v", newHosts)
		hosts = newHosts
	}

	if user != "" {
		newHosts := make([]*Host, 0)
		for _, host := range hosts {
			if host.Username == user {
				newHosts = append(newHosts, host)
			}
		}
		logger.Debugf("Filtered hosts by user: %+#v", newHosts)
		hosts = newHosts
	}

	if tags != "" {
		tagsMap := make(map[string]bool)
		for _, tag := range strings.Split(tags, ",") {
			tagsMap[tag] = true
		}

		newHosts := make([]*Host, 0)
		for _, host := range hosts {
			for _, tag := range host.TagList {
				if tagsMap[tag] {
					newHosts = append(newHosts, host)
					break
				}
			}
		}
		logger.Debugf("Filtered hosts by tags: %+#v", newHosts)
		hosts = newHosts
	}

	return hosts, err
}

func ListConfigHosts(name string, user string, tags string) error {
	w := tabwriter.NewWriter(os.Stdout, 12, 8, 4, ' ', 0)
	fmt.Fprintln(w, "PATTERNS\tHOST\tUSER\tPORT\tJUMP\tTAGS\t")
	hosts, err := FilteredHosts(name, user, tags)
	if err != nil {
		return err
	}
	for _, host := range hosts {
		row := fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t",
			strings.Join(host.Patterns, ","),
			host.HostName,
			host.Username,
			host.Port,
			host.ProxyJump,
			strings.Join(host.TagList, ","),
		)
		fmt.Fprintln(w, row)
	}
	w.Flush()
	return nil
}

func saveConfig() error {
	if data, err := yaml.Marshal(Config); err != nil {
		return err
	} else {
		return os.WriteFile(viper.ConfigFileUsed(), data, 0644)
	}
}
