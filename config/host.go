package config

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	HOST_LIST_HEADERS = "NAME\tHOST\tUSER\tPORT\tJUMP\tTAGS\tTIMEOUT\t"
)

type Host struct {
	Addr     string   `yaml:"addr"`
	Jump     string   `yaml:"jump,omitempty"`
	Tags     string   `yaml:"tags,omitempty"`
	Timeout  int      `yaml:"timeout,omitempty"`
	Name     string   `yaml:"-"`
	User     string   `yaml:"-"`
	Host     string   `yaml:"-"`
	Port     uint16   `yaml:"-"`
	TagList  []string `yaml:"-"`
	JumpList []*Host  `yaml:"-"`
}

func (host *Host) EndPoint() string {
	if host.Port == 0 {
		return host.Host
	}
	return fmt.Sprintf("%v:%v", host.Host, host.Port)
}

func (host *Host) String() string {
	if host.User == "" {
		return host.EndPoint()
	}
	return fmt.Sprintf("%v@%v", host.User, host.EndPoint())
}

func (host *Host) JumpString() string {
	hosts := make([]string, 0)
	for _, host := range host.JumpList {
		hosts = append(hosts, host.String())
	}
	return strings.Join(hosts, ",")
}

func (host *Host) Row() string {
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\t%d\t",
		host.Name, host.Host, host.User, host.Port, host.Jump, host.Tags, host.Timeout)
}

func (host *Host) SetDefaultValue() {
	if host.User == "" {
		if Config.DefaultUser != "" {
			host.User = Config.DefaultUser
		} else {
			host.User = "root"
		}
	}

	if host.Port == 0 {
		if Config.DefaultPort > 0 {
			host.Port = Config.DefaultPort
		} else {
			host.Port = 22
		}
	}

	if host.Jump == "" && Config.DefaultJump != "" {
		host.Jump = Config.DefaultJump
		host.JumpList = Config.JumpHosts
	}

	if host.Timeout < 0 {
		if Config.DefaultTimeout >= 0 {
			host.Timeout = Config.DefaultTimeout
		} else {
			host.Timeout = 0
		}
	}
}

func (host *Host) SetOverlayValue() {
	host.SetDefaultValue()

	if Config.OverlayUser != "" {
		host.User = Config.OverlayUser
	}
	if Config.OverlayPort > 0 {
		host.Port = Config.OverlayPort
	}
	if Config.OverlayTimeout >= 0 {
		host.Timeout = Config.OverlayTimeout
	}
	if Config.OverlayJump != "" {
		host.Jump = Config.OverlayJump
		host.JumpList = Config.JumpHosts
	}
	if host.Jump == "" && Config.DefaultJump != "" {
		host.Jump = Config.DefaultJump
		host.JumpList = Config.JumpHosts
	}
}

func (host *Host) TagsFormat() {
	var tags []string
	for _, tag := range strings.Split(host.Tags, ",") {
		if tag != "" && tag != "all" {
			existed := false
			for _, tag_ := range tags {
				if tag == tag_ {
					existed = true
					break
				}
			}
			if existed == false {
				tags = append(tags, tag)
			}
		}
	}
	host.Tags = strings.Join(sort.StringSlice(tags), ",")
}

func (host *Host) parse(isJump bool) (err error) {
	// user, host
	if parts := strings.Split(host.Addr, "@"); len(parts) > 2 {
		return fmt.Errorf("Invalid host format: %v", host.Addr)
	} else if len(parts) == 2 {
		host.User = parts[0]
		host.Host = parts[1]
	} else {
		host.Host = host.Addr
	}

	if host.Host == "" {
		return fmt.Errorf("Host required non-empty string.")
	}

	// port
	if parts := strings.Split(host.Host, ":"); len(parts) > 2 {
		return fmt.Errorf("Invalid host format: %v", host.Addr)
	} else if len(parts) == 2 {
		host.Host = parts[0]
		if port, err := strconv.Atoi(parts[1]); err != nil {
			return err
		} else if port <= 0 || port >= 65536 {
			return fmt.Errorf("Invalid host format: %v", host.Addr)
		} else {
			host.Port = uint16(port)
		}
	}

	// tags
	if err := CheckTags(host.Tags); err != nil {
		return err
	}
	for _, tag := range strings.Split(host.Tags, ",") {
		if len(tag) > 0 {
			host.TagList = append(host.TagList, tag)
		}
	}

	// jump
	if !isJump {
		if host.Jump != "" && host.Jump != "none" {
			for _, hostString := range strings.Split(host.Jump, ",") {
				if len(hostString) == 0 {
					continue
				}
				jumpHost := &Host{Addr: hostString}
				if err := jumpHost.parse(true); err != nil {
					return err
				}
				if len(jumpHost.JumpList) > 0 {
					return fmt.Errorf("host \"%s\" jump is nested!", host.Name)
				}
				host.JumpList = append(host.JumpList, jumpHost)
			}
		}
	}

	if isJump {
		host.SetDefaultValue()
	}
	return err
}

func (host *Host) Parse() (err error) {
	return host.parse(false)
}

func CheckTags(tags string) error {
	if matched, err := regexp.MatchString(`[0-9a-zA-z_\-,]*`, tags); err != nil {
		return err
	} else if !matched {
		return fmt.Errorf("Invalid tags!")
	}
	return nil
}

func AddHost(name string, user string, ip string, port uint16, jump string, tags string, timeout int) error {
	if name == "" {
		return fmt.Errorf("required name is non-empty string!")
	}
	if _, exist := Config.Hosts[name]; exist {
		return fmt.Errorf("host name \"%s\" already existed!", name)
	}
	host := &Host{Name: name, User: user, Host: ip, Port: port, Jump: jump, Tags: tags, Timeout: timeout}
	host.Addr = host.String()
	host.TagsFormat()
	if host.Timeout < 0 {
		host.Timeout = 0
	}
	Config.Hosts[name] = host
	return saveConfig()
}

func UpdateHost(name string, user string, ip string, port uint16, jump string, tags string, timeout int) error {
	if name == "" {
		return fmt.Errorf("required name is non-empty string!")
	}
	if _, exist := Config.Hosts[name]; exist {
		return fmt.Errorf("host name \"%s\" already existed!", name)
	}
	host := &Host{Name: name}
	if user != "" {
		host.User = user
	}
	if ip != "" {
		host.Host = ip
	}
	if port != 0 {
		host.Port = 22
	}
	if jump != "" {
		host.Jump = jump
	}
	if tags != "" {
		host.Tags = tags
	}
	if timeout >= 0 {
		host.Timeout = timeout
	}
	host.Addr = host.String()
	host.TagsFormat()
	Config.Hosts[name] = host
	return saveConfig()
}

func DeleteHost(name string) error {
	if name == "" {
		return fmt.Errorf("required name is non-empty string!")
	}
	if _, exist := Config.Hosts[name]; !exist {
		return fmt.Errorf("host name \"%s\" is not exists!", name)
	}
	delete(Config.Hosts, name)
	return saveConfig()
}

func GetHostNames() (names []string) {
	for name := range Config.Hosts {
		names = append(names, name)
	}
	return names
}
