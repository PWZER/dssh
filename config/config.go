package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

type ConfigType struct {
	ModulesDir     string           `yaml:"modulesDir,omitempty"`
	SSHAuthSock    string           `yaml:"sshAuthSock,omitempty"`
	DefaultTimeout int              `yaml:"defaultTimeout,omitempty"`
	DefaultUser    string           `yaml:"defaultUser,omitempty"`
	DefaultPort    uint16           `yaml:"defaultPort,omitempty"`
	DefaultJump    string           `yaml:"defaultJump,omitempty"`
	Hosts          map[string]*Host `yaml:"hosts"`
	Parallel       int              `yaml:"-"`
	OverlayTimeout int              `yaml:"-"`
	OverlayUser    string           `yaml:"-"`
	OverlayPort    uint16           `yaml:"-"`
	OverlayJump    string           `yaml:"-"`
	OverlayHost    string           `yaml:"-"`
	JumpHosts      []*Host          `yaml:"-"`
}

var Config = &ConfigType{Parallel: 1, OverlayTimeout: -1}

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

	for name, host := range Config.Hosts {
		host.Name = name
		host.TagsFormat()
		if err := host.Parse(); err != nil {
			return err
		}
	}

	jump := Config.DefaultJump
	if Config.OverlayJump != "" {
		jump = Config.OverlayJump
	}
	if jump != "" && jump != "none" {
		for _, hostString := range strings.Split(jump, ",") {
			host := &Host{Addr: hostString}
			if err := host.parse(true); err != nil {
				return err
			}
			Config.JumpHosts = append(Config.JumpHosts, host)
		}
	}
	return nil
}

func ConfigHostsFilter(name string, user string, tags string) (hosts []*Host, err error) {
	if name != "" {
		// name filter
		if host, ok := Config.Hosts[name]; ok {
			hosts = append(hosts, host)
		}
		return hosts, err
	}

	var hostNames []string
	for hostName := range Config.Hosts {
		hostNames = append(hostNames, hostName)
	}
	sort.Strings(hostNames)
	for _, name := range hostNames {
		host, ok := Config.Hosts[name]
		if !ok {
			continue
		}

		// user filter
		if user != "" && host.User != user {
			continue
		}

		// tags filter
		if tags != "" {
			if matched, err := regexp.MatchString(`[0-9a-zA-z_\-,]*`, tags); err != nil {
				return hosts, err
			} else if !matched {
				return hosts, fmt.Errorf("Invalid tags!")
			}

			hasTags := false
			for _, tag := range strings.Split(tags, ",") {
				if tag == "all" || strings.Contains(host.Tags, tag) {
					hasTags = true
					break
				}
			}
			if !hasTags {
				continue
			}
		}

		hosts = append(hosts, host)
	}
	return hosts, err
}

func ListConfigHosts(name string, user string, tags string) error {
	w := tabwriter.NewWriter(os.Stdout, 12, 8, 4, ' ', 0)
	fmt.Fprintln(w, HOST_LIST_HEADERS)
	hosts, err := ConfigHostsFilter(name, user, tags)
	if err != nil {
		return err
	}
	for _, host := range hosts {
		fmt.Fprintln(w, host.Row())
	}
	w.Flush()
	return nil
}

func saveConfig() error {
	if data, err := yaml.Marshal(Config); err != nil {
		return err
	} else {
		return ioutil.WriteFile(viper.ConfigFileUsed(), data, 0644)
	}
}
