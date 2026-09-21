package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"

	"github.com/PWZER/dssh/logger"
	"github.com/PWZER/dssh/utils"
)

var knownhostsErrLineRe = regexp.MustCompile(`knownhosts: .+?:(\d+):`)

func knownHostsFiles() []string {
	paths := []string{}
	paths = append(paths, filepath.Join(utils.HomeDir(), ".ssh", "known_hosts"))
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
	return filepath.Join(utils.HomeDir(), ".ssh", "known_hosts")
}

// usableRemoteAddr returns the remote address string when it carries real
// peer information. Connections tunneled through a jump host carry a zero
// address ("0.0.0.0:0"), which must never be displayed or recorded.
func usableRemoteAddr(remote net.Addr) string {
	if remote == nil {
		return ""
	}
	addr := remote.String()
	if tcpAddr, ok := remote.(*net.TCPAddr); ok {
		if tcpAddr.IP == nil || tcpAddr.IP.IsUnspecified() || tcpAddr.Port == 0 {
			return ""
		}
		return addr
	}
	if addr == "" || addr == "0.0.0.0:0" {
		return ""
	}
	return addr
}

// sanitizeKnownHosts rewrites each known_hosts file into a temporary file
// containing only parseable lines, OpenSSH style: malformed lines are
// skipped instead of failing every connection. Returns the temp file paths.
func sanitizeKnownHosts(paths []string) ([]string, error) {
	sanitized := make([]string, 0, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warnf("ignore unreadable known_hosts file %s: %v", path, err)
			continue
		}

		lines := strings.Split(string(content), "\n")
		// drop malformed lines until the file parses cleanly
		for {
			// knownhosts.New does not expose its db; validate by loading
			// the current line set through a temp file
			tmpValidate, validateErr := os.CreateTemp("", "dssh-known_hosts-validate-*")
			if validateErr != nil {
				return nil, validateErr
			}
			if _, writeErr := tmpValidate.WriteString(strings.Join(lines, "\n")); writeErr != nil {
				tmpValidate.Close()
				return nil, writeErr
			}
			tmpValidate.Close()
			_, dbErr := knownhosts.New(tmpValidate.Name())
			os.Remove(tmpValidate.Name())
			if dbErr == nil {
				break
			}
			// error format: "knownhosts: <file>:<line>: <detail>"
			lineNum := 0
			if matches := knownhostsErrLineRe.FindStringSubmatch(dbErr.Error()); len(matches) > 1 {
				fmt.Sscanf(matches[1], "%d", &lineNum)
			}
			if lineNum <= 0 || lineNum > len(lines) {
				// cannot locate the bad line, give up on this file
				logger.Warnf("ignore malformed known_hosts file %s: %v", path, dbErr)
				lines = nil
				break
			}
			logger.Warnf("skipped malformed line %d in known_hosts file %s", lineNum, path)
			lines = append(lines[:lineNum-1], lines[lineNum:]...)
		}
		if len(lines) == 0 {
			continue
		}

		tmpFile, err := os.CreateTemp("", "dssh-known_hosts-*")
		if err != nil {
			return nil, fmt.Errorf("create temp known_hosts file: %w", err)
		}
		if _, err := tmpFile.WriteString(strings.Join(lines, "\n")); err != nil {
			tmpFile.Close()
			return nil, fmt.Errorf("write temp known_hosts file: %w", err)
		}
		tmpFile.Close()
		sanitized = append(sanitized, tmpFile.Name())
	}
	return sanitized, nil
}

// newKnownHostsChecker loads the known_hosts files into a checker. Unlike
// knownhosts.New, malformed lines are skipped (OpenSSH style) instead of
// failing every connection.
func newKnownHostsChecker(paths []string) (gossh.HostKeyCallback, error) {
	sanitized, err := sanitizeKnownHosts(paths)
	if err != nil {
		return nil, err
	}
	checker, err := knownhosts.New(sanitized...)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts: %w", err)
	}
	return checker, nil
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
	checker, checkerErr := newKnownHostsChecker(knownHostsFiles())
	if checkerErr != nil {
		return checkerErr
	}
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
