package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	"github.com/PWZER/dssh/logger"
)

// homeDir returns the current user's home directory.
func homeDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return dir
}

// knownHostsFiles returns the existing known_hosts file paths, the user file
// takes precedence over the system one. Missing files are not an error, an
// empty list means every host key is unknown.
func knownHostsFiles() []string {
	paths := []string{}
	paths = append(paths, filepath.Join(homeDir(), ".ssh", "known_hosts"))
	paths = append(paths, "/etc/ssh/ssh_known_hosts")

	existing := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			existing = append(existing, path)
		}
	}
	return existing
}

// userKnownHostsFile returns the writable user known_hosts file path.
func userKnownHostsFile() string {
	return filepath.Join(homeDir(), ".ssh", "known_hosts")
}

// usableRemoteAddr returns the remote address string when it carries real
// peer information. Connections tunneled through a jump host carry a zero
// address ("0.0.0.0:0"), which must never be displayed or recorded.
func usableRemoteAddr(remote net.Addr) string {
	tcpAddr, ok := remote.(*net.TCPAddr)
	if !ok {
		if remote != nil && remote.String() != "" && remote.String() != "0.0.0.0:0" {
			return remote.String()
		}
		return ""
	}
	if tcpAddr.IP == nil || tcpAddr.IP.IsUnspecified() || tcpAddr.Port == 0 {
		return ""
	}
	return tcpAddr.String()
}

// newKnownHostsChecker loads the known_hosts files into a checker. Unlike
// knownhosts.New, one malformed file degrades to being skipped (OpenSSH
// skips malformed lines) instead of failing every connection.
func newKnownHostsChecker(paths []string) gossh.HostKeyCallback {
	for {
		checker, err := knownhosts.New(paths...)
		if err == nil {
			return checker
		}
		dropped := false
		for i, path := range paths {
			if strings.Contains(err.Error(), path) {
				logger.Warnf("ignore malformed known_hosts file %s: %v", path, err)
				paths = append(paths[:i], paths[i+1:]...)
				dropped = true
				break
			}
		}
		if !dropped {
			// no file to drop: treat every host key as unknown (TOFU)
			logger.Warnf("ignore known_hosts load error, all host keys will be treated as unknown: %v", err)
			return func(host string, remote net.Addr, key gossh.PublicKey) error {
				return &knownhosts.KeyError{}
			}
		}
	}
}

// confirmHostKey prompts the user to trust an unknown host key, OpenSSH style.
func confirmHostKey(host, remoteAddr string, key gossh.PublicKey) bool {
	fmt.Printf("The authenticity of host '%s' can't be established.\n", host)
	if remoteAddr != "" {
		fmt.Printf("... but the remote address is '%s'.\n", remoteAddr)
	}
	fmt.Printf("%s key fingerprint is %s.\n", key.Type(), gossh.FingerprintSHA256(key))
	fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "yes" || answer == "y"
}

// appendKnownHost appends the verified host key to the user known_hosts file
// under the verified hostname (and its resolved address when available), so
// the entry matches on the next connection.
func appendKnownHost(host, remoteAddr string, key gossh.PublicKey) error {
	addresses := []string{knownhosts.Normalize(host)}
	if remoteAddr != "" {
		if normalized := knownhosts.Normalize(remoteAddr); normalized != addresses[0] {
			addresses = append(addresses, normalized)
		}
	}

	path := userKnownHostsFile()
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	line := knownhosts.Line(addresses, key)
	_, err = fmt.Fprintln(file, line)
	return err
}

// HostKeyCallback verifies the server host key against known_hosts,
// OpenSSH style: known keys are checked strictly, unknown hosts are
// confirmed interactively and appended to the user known_hosts file,
// changed keys are rejected with a warning.
func HostKeyCallback(host string, remote net.Addr, key gossh.PublicKey) error {
	checker := newKnownHostsChecker(knownHostsFiles())
	err := checker(host, remote, key)
	if err == nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if !errors.As(err, &keyErr) {
		return err
	}

	remoteAddr := usableRemoteAddr(remote)
	if len(keyErr.Want) > 0 {
		fmt.Fprintf(os.Stderr, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
		fmt.Fprintf(os.Stderr, "@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @\n")
		fmt.Fprintf(os.Stderr, "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@\n")
		fmt.Fprintf(os.Stderr, "IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n")
		return fmt.Errorf("host key for %s has changed and could not be verified: %w", host, err)
	}

	// unknown host key
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("host key for %s is unknown and stdin is not a terminal", host)
	}
	if !confirmHostKey(host, remoteAddr, key) {
		return fmt.Errorf("host key for %s was rejected by user", host)
	}
	if err := appendKnownHost(host, remoteAddr, key); err != nil {
		return fmt.Errorf("failed to add host key to known_hosts: %w", err)
	}
	fmt.Println("Warning: Permanently added host key to the list of known hosts.")
	return nil
}
