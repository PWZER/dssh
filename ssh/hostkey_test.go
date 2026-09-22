package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestSanitizeKnownHosts(t *testing.T) {
	// isolate TMPDIR so leak assertions are immune to files from other
	// processes or stale leftovers
	t.Setenv("TMPDIR", t.TempDir())

	// generate a valid host key line
	key, _, _, _, err := gossh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGPKfpLL60VzCZYsDrT+jJ0Sd7cnEyv7ipHdOM0FtR1p test@example"))
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	validLine := "127.0.0.1 " + strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))

	tests := []struct {
		name     string
		content  string
		wantTemp bool // expect a sanitized temp file back
		wantErr  bool
	}{
		{"empty file", "", false, false},
		{"valid line", validLine + "\n", false, false},
		{"valid line without newline", validLine, false, false},
		{"valid + garbage", validLine + "\ngarbage line here\n", true, false},
		{"all garbage", "garbage\nmore garbage\n", false, true},
		{"all garbage without newline", "garbage", false, true},
		{"garbage + blank lines", "garbage\n\n", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempsBefore := tempKnownHostsFiles()

			tmp, err := os.CreateTemp("", "dssh-test-kh-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmp.Name())
			tmp.WriteString(tt.content)
			tmp.Close()

			sanitized, err := sanitizeKnownHosts(tmp.Name())
			if (err != nil) != tt.wantErr {
				t.Fatalf("sanitizeKnownHosts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				// no temp file may leak on error paths
				if leaked := tempKnownHostsFiles(); len(leaked) != len(tempsBefore) {
					t.Fatalf("temp file leaked on error path: %v", leaked)
				}
				return
			}
			if (sanitized != "") != tt.wantTemp {
				t.Fatalf("sanitizeKnownHosts() = %q, wantTemp %v", sanitized, tt.wantTemp)
			}
			if sanitized == "" {
				return
			}
			// the sanitized file must parse cleanly, remove it afterwards
			defer os.Remove(sanitized)
			if _, err := knownhosts.New(sanitized); err != nil {
				t.Fatalf("sanitized file %s does not parse: %v", sanitized, err)
			}
		})
	}
}

func TestNewKnownHostsChecker(t *testing.T) {
	// isolate TMPDIR so leak assertions are immune to files from other
	// processes or stale leftovers
	t.Setenv("TMPDIR", t.TempDir())

	key, _, _, _, err := gossh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGPKfpLL60VzCZYsDrT+jJ0Sd7cnEyv7ipHdOM0FtR1p test@example"))
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	validLine := "127.0.0.1 " + strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))

	tmp, err := os.CreateTemp("", "dssh-test-kh-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString(validLine + "\ngarbage line here\n")
	tmp.Close()

	checker, err := newKnownHostsChecker([]string{tmp.Name()})
	if err != nil {
		t.Fatalf("newKnownHostsChecker() error = %v", err)
	}
	if checker == nil {
		t.Fatal("newKnownHostsChecker() returned nil checker")
	}
	// temp copies are removed before returning, the isolated TMPDIR
	// starts empty so nothing may remain
	if leaked := tempKnownHostsFiles(); len(leaked) != 0 {
		t.Fatalf("temp files leaked: %v", leaked)
	}
}

func tempKnownHostsFiles() []string {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dssh-known_hosts-*"))
	return matches
}
