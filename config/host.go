package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"

	"github.com/PWZER/dssh/logger"
	"github.com/PWZER/dssh/utils"
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
	return newHost(username, hostname, port, proxyJump, identityFiles, nil)
}

func newHost(username, hostname string, port uint16, proxyJump string, identityFiles []string, jumpChain []string) (host *Host, err error) {
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

	// parse format hostname:port (IPv6 aware)
	host.HostName, host.Port, err = parseHostPort(host.HostName, host.Port)
	if err != nil {
		return nil, err
	}

	// identity files
	for _, identityFile := range identityFiles {
		identityFile = expandHomePath(identityFile)
		if _, err := os.Stat(identityFile); err != nil {
			continue
		}
		host.IdentityFiles = append(host.IdentityFiles, identityFile)
	}

	if err = host.fillSSHConfigAttrs(jumpChain); err != nil {
		return nil, err
	}

	logger.Debugf("host: %+#v", host)
	return host, nil
}

// parseHostPort parses the "hostname:port" format, IPv6 addresses are
// supported: "::1" (bare address), "[::1]" and "[::1]:22". The port argument
// is the already set port (0 when unset), it is kept as is when the input
// has no port part.
func parseHostPort(hostname string, port uint16) (string, uint16, error) {
	if !strings.Contains(hostname, ":") {
		return hostname, port, nil
	}

	// bracketed IPv6: "[addr]" or "[addr]:port"
	if strings.HasPrefix(hostname, "[") {
		idx := strings.LastIndex(hostname, "]")
		if idx < 0 {
			return "", 0, fmt.Errorf("invalid hostname format: %v", hostname)
		}
		addr := hostname[1:idx]
		rest := hostname[idx+1:]
		if rest == "" {
			return addr, port, nil
		}
		if !strings.HasPrefix(rest, ":") {
			return "", 0, fmt.Errorf("invalid hostname format: %v", hostname)
		}
		if port != 0 {
			return "", 0, fmt.Errorf("port is already set: %v", hostname)
		}
		portInt, err := strconv.Atoi(rest[1:])
		if err != nil || portInt <= 0 || portInt >= 65536 {
			return "", 0, fmt.Errorf("invalid port format: %v", rest[1:])
		}
		return addr, uint16(portInt), nil
	}

	// bare IPv6 address (multiple colons) has no port part
	if strings.Count(hostname, ":") > 1 {
		if net.ParseIP(hostname) == nil {
			return "", 0, fmt.Errorf("invalid hostname format: %v", hostname)
		}
		return hostname, port, nil
	}

	// hostname:port
	idx := strings.LastIndex(hostname, ":")
	addr, portPart := hostname[:idx], hostname[idx+1:]
	if port != 0 {
		return "", 0, fmt.Errorf("port is already set: %v", hostname)
	}
	portInt, err := strconv.Atoi(portPart)
	if err != nil || portInt <= 0 || portInt >= 65536 {
		return "", 0, fmt.Errorf("invalid port format: %v", portPart)
	}
	return addr, uint16(portInt), nil
}

// expandHomePath expands a leading "~" or "~/" to the user's home directory.
func expandHomePath(path string) string {
	if path == "~" {
		return utils.HomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(utils.HomeDir(), path[2:])
	}
	return path
}

func (host *Host) EndPoint() string {
	if host.Port == 0 {
		return host.HostName
	}
	if strings.Contains(host.HostName, ":") {
		// IPv6 address needs brackets around it
		return fmt.Sprintf("[%v]:%v", host.HostName, host.Port)
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
		rawPort := ssh_config.Get(host.HostName, "Port")
		portInt, err := strconv.Atoi(rawPort)
		if err != nil {
			if rawPort != "" {
				logger.Warnf("invalid port in ssh config: %v", rawPort)
			}
		} else if portInt <= 0 || portInt >= 65536 {
			logger.Warnf("invalid port in ssh config: %v", rawPort)
		} else {
			host.Port = uint16(portInt)
		}
	}

	// default port
	if host.Port == 0 {
		host.Port = 22
	}
}

func (host *Host) fillProxyJump(jumpChain []string) error {
	if host.ProxyJump == "" {
		// fill proxy jump with patterns
		for _, pattern := range host.Patterns {
			if strings.ContainsAny(pattern, "*!?") {
				continue
			}
			host.ProxyJump = ssh_config.Get(pattern, "ProxyJump")
			if host.ProxyJump != "" {
				break
			}
		}

		// fill proxy jump with host name
		if host.ProxyJump == "" {
			host.ProxyJump = ssh_config.Get(host.HostName, "ProxyJump")
		}
	}

	// jump list
	if host.ProxyJump != "" {
		jumps, err := buildJumpChain(host.ProxyJump, jumpChain, host.IdentityFiles)
		if err != nil {
			return err
		}
		host.JumpList = jumps
	}
	return nil
}

// buildJumpChain builds the flattened connection-ordered jump chain for the
// given ProxyJump value. jumpChain carries the jumps being visited up the
// recursion to detect cycles. Invalid jumps are reported as errors instead
// of being silently skipped.
func buildJumpChain(proxyJump string, jumpChain []string, identityFiles []string) ([]*Host, error) {
	return buildJumpChainWith(proxyJump, jumpChain, func(jump string, chain []string) (*Host, error) {
		return newHost("", jump, 0, "", identityFiles, chain)
	})
}

func buildJumpChainWith(proxyJump string, jumpChain []string, resolveJump func(jump string, chain []string) (*Host, error)) ([]*Host, error) {
	chain := make([]*Host, 0)
	for _, jump := range strings.Split(proxyJump, ",") {
		if jump == "" {
			continue
		}
		if slices.Contains(jumpChain, jump) {
			return nil, fmt.Errorf("proxy jump cycle detected: %v", jump)
		}
		jumpHost, err := resolveJump(jump, append(slices.Clone(jumpChain), jump))
		if err != nil {
			return nil, err
		}
		chain = append(chain, jumpHost)
	}
	chain = flattenJumpList(chain)
	// normalized check catches the same host spelled differently ("a" vs
	// "user@a" vs "a:22") appearing twice in one jump chain
	seen := make(map[string]bool, len(chain))
	for _, jumpHost := range chain {
		endPoint := jumpHost.EndPoint()
		if seen[endPoint] {
			return nil, fmt.Errorf("duplicate jump host: %v", jumpHost.Summary())
		}
		seen[endPoint] = true
	}
	return chain, nil
}

// flattenJumpList expands nested jump chains into a flat connection-ordered
// list: each jump's own nested jumps come before the jump itself, e.g.
// target -> a (a -> b) is flattened to [b, a].
func flattenJumpList(jumps []*Host) []*Host {
	flattened := make([]*Host, 0, len(jumps))
	for _, jump := range jumps {
		flattened = append(flattened, flattenJumpList(jump.JumpList)...)
		jump.JumpList = nil
		flattened = append(flattened, jump)
	}
	return flattened
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
			identityFile = expandHomePath(identityFile)
			if _, err := os.Stat(identityFile); err == nil {
				host.IdentityFiles = append(host.IdentityFiles, identityFile)
			}
		}
	}

	// fill identity files with host name
	if len(host.IdentityFiles) == 0 {
		identityFiles := ssh_config.GetAll(host.HostName, "IdentityFile")
		for _, identityFile := range identityFiles {
			identityFile = expandHomePath(identityFile)
			if _, err := os.Stat(identityFile); err == nil {
				host.IdentityFiles = append(host.IdentityFiles, identityFile)
			}
		}
	}

	// default identity file
	if len(host.IdentityFiles) == 0 {
		defaultIdentityFile := filepath.Join(utils.HomeDir(), ".ssh", "id_rsa")
		if _, err := os.Stat(defaultIdentityFile); err == nil {
			host.IdentityFiles = []string{defaultIdentityFile}
		}
	}
}

func (host *Host) FillAttrsWithSSHConfig() error {
	return host.fillSSHConfigAttrs(nil)
}

func (host *Host) fillSSHConfigAttrs(jumpChain []string) error {
	host.fillUsername()
	host.fillPort()
	host.fillIdentityFiles()
	if err := host.fillProxyJump(jumpChain); err != nil { // must after identity files
		return err
	}

	rawHostname := ssh_config.Get(host.HostName, "HostName")
	if rawHostname != "" {
		host.HostName = rawHostname
	}
	return nil
}
