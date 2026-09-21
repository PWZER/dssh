package ssh

import (
	"os"
	"strings"
	"testing"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestSanitizeKnownHosts(t *testing.T) {
	// generate a valid host key line
	key, _, _, _, err := gossh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGPKfpLL60VzCZYsDrT+jJ0Sd7cnEyv7ipHdOM0FtR1p test@example"))
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	validLine := "127.0.0.1 " + key.Type() + " " + strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		wantSkip bool // expect at least one malformed line to be skipped
	}{
		{"empty file", "", false, false},
		{"valid line", validLine, false, false},
		{"valid + garbage", validLine + "\ngarbage line here\n", false, true},
		{"all garbage", "garbage\nmore garbage\n", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp, err := os.CreateTemp("", "dssh-test-kh-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmp.Name())
			tmp.WriteString(tt.content)
			tmp.Close()

			sanitized, err := sanitizeKnownHosts([]string{tmp.Name()})
			if (err != nil) != tt.wantErr {
				t.Fatalf("sanitizeKnownHosts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(sanitized) == 0 {
				// empty or all-garbage content produces no sanitized file
				if tt.content == "" || strings.Count(tt.content, "garbage") == strings.Count(tt.content, "\n") {
					return
				}
				t.Fatalf("sanitizeKnownHosts() returned no files for non-empty input")
			}
			for _, path := range sanitized {
				defer os.Remove(path)
				// verify the sanitized file parses cleanly
				if _, err := knownhosts.New(path); err != nil {
					t.Fatalf("sanitized file %s does not parse: %v", path, err)
				}
			}
		})
	}
}
