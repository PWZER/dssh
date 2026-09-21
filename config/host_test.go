package config

import (
	"os"
	"path/filepath"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
)

func TestMain(m *testing.M) {
	// keep t.Setenv("HOME") effective, go-homedir caches the result
	homedir.DisableCache = true
	os.Exit(m.Run())
}

func TestParseHostPort(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		port     uint16
		wantHost string
		wantPort uint16
		wantErr  bool
	}{
		{"plain host", "example.com", 0, "example.com", 0, false},
		{"plain host with preset port", "example.com", 2222, "example.com", 2222, false},
		{"host with port", "example.com:2222", 0, "example.com", 2222, false},
		{"host with port conflict", "example.com:2222", 3333, "", 0, true},
		{"host with invalid port", "example.com:0", 0, "", 0, true},
		{"host with out of range port", "example.com:65536", 0, "", 0, true},
		{"host with non numeric port", "example.com:foo", 0, "", 0, true},
		{"host with empty port", "example.com:", 0, "", 0, true},
		{"bare ipv6", "::1", 0, "::1", 0, false},
		{"bare ipv6 with preset port", "::1", 2222, "::1", 2222, false},
		{"bracketed ipv6 with port", "[::1]:22", 0, "::1", 22, false},
		{"bracketed ipv6 without port", "[fe80::1]", 0, "fe80::1", 0, false},
		{"bracketed ipv6 with preset port", "[fe80::1]", 2222, "fe80::1", 2222, false},
		{"bracketed ipv6 with port conflict", "[::1]:22", 2222, "", 0, true},
		{"bracketed ipv6 with invalid port", "[::1]:foo", 0, "", 0, true},
		{"unterminated bracket", "[::1:22", 0, "", 0, true},
		{"bare ipv6 with zone", "fe80::1%eth0", 0, "fe80::1%eth0", 0, false},
		{"bare ipv6 with zone and preset port", "fe80::1%eth0", 2222, "fe80::1%eth0", 2222, false},
		{"invalid multi-colon", "example.com:2222:3333", 0, "", 0, true},
		{"invalid multi-colon typo", "host:a:b", 0, "", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotHost, gotPort, err := parseHostPort(c.input, c.port)
			if c.wantErr {
				if err == nil {
					t.Fatalf("parseHostPort(%q, %d) expected error, got host=%q port=%d", c.input, c.port, gotHost, gotPort)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseHostPort(%q, %d) unexpected error: %v", c.input, c.port, err)
			}
			if gotHost != c.wantHost || gotPort != c.wantPort {
				t.Fatalf("parseHostPort(%q, %d) = (%q, %d), want (%q, %d)", c.input, c.port, gotHost, gotPort, c.wantHost, c.wantPort)
			}
		})
	}
}

func TestParseHostPortIPv6Endpoint(t *testing.T) {
	host := &Host{HostName: "::1", Port: 22}
	if got := host.EndPoint(); got != "[::1]:22" {
		t.Fatalf("EndPoint() = %q, want %q", got, "[::1]:22")
	}
	host = &Host{HostName: "example.com", Port: 2222}
	if got := host.EndPoint(); got != "example.com:2222" {
		t.Fatalf("EndPoint() = %q, want %q", got, "example.com:2222")
	}
}

func TestExpandHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		input string
		want  string
	}{
		{"~", home},
		{"~/.ssh/id_rsa", filepath.Join(home, ".ssh", "id_rsa")},
		{"/etc/hosts", "/etc/hosts"},
		{"relative/path", "relative/path"},
	}
	for _, c := range cases {
		if got := expandHomePath(c.input); got != c.want {
			t.Fatalf("expandHomePath(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFlattenJumpList(t *testing.T) {
	b := &Host{HostName: "b"}
	c := &Host{HostName: "c"}
	a := &Host{HostName: "a", JumpList: []*Host{b}}
	target := &Host{HostName: "target", JumpList: []*Host{c, a}}

	flat := flattenJumpList(target.JumpList)

	// connection order: nested jumps come before the jump itself,
	// target -> [c, a] with a -> b is flattened to [c, b, a]
	if len(flat) != 3 {
		t.Fatalf("flattenJumpList len = %d, want 3", len(flat))
	}
	wantOrder := []string{"c", "b", "a"}
	for i, want := range wantOrder {
		if flat[i].HostName != want {
			t.Fatalf("flattenJumpList[%d] = %q, want %q", i, flat[i].HostName, want)
		}
	}
	if a.JumpList != nil {
		t.Fatalf("nested jump list of a should be cleared")
	}
}

func TestBuildJumpChainWithCycle(t *testing.T) {
	// resolveJump mimics newHost: the child host resolves its own
	// ProxyJump and recursively builds its own chain.
	var resolve func(jump string, chain []string) (*Host, error)
	resolve = func(jump string, chain []string) (*Host, error) {
		h := &Host{HostName: jump}
		switch jump {
		case "a":
			h.ProxyJump = "b"
		case "b":
			h.ProxyJump = "a"
		}
		var err error
		h.JumpList, err = buildJumpChainWith(h.ProxyJump, chain, resolve)
		return h, err
	}

	if _, err := buildJumpChainWith("a", nil, resolve); err == nil {
		t.Fatal("expected cycle error for a <-> b")
	}

	var resolveSelf func(jump string, chain []string) (*Host, error)
	resolveSelf = func(jump string, chain []string) (*Host, error) {
		h := &Host{HostName: jump, ProxyJump: jump}
		var err error
		h.JumpList, err = buildJumpChainWith(h.ProxyJump, chain, resolveSelf)
		return h, err
	}
	if _, err := buildJumpChainWith("a", nil, resolveSelf); err == nil {
		t.Fatal("expected cycle error for a -> a")
	}
}

func TestBuildJumpChainWithNested(t *testing.T) {
	// a -> b, target -> a => flattened chain [b, a]
	var resolve func(jump string, chain []string) (*Host, error)
	resolve = func(jump string, chain []string) (*Host, error) {
		h := &Host{HostName: jump}
		if jump == "a" {
			h.ProxyJump = "b"
		}
		var err error
		h.JumpList, err = buildJumpChainWith(h.ProxyJump, chain, resolve)
		return h, err
	}

	chain, err := buildJumpChainWith("a", nil, resolve)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 2 || chain[0].HostName != "b" || chain[1].HostName != "a" {
		t.Fatalf("chain = %v, want [b a]", hostNames(chain))
	}
	if chain[0].JumpList != nil || chain[1].JumpList != nil {
		t.Fatal("nested jump lists should be cleared after flattening")
	}
}

func TestBuildJumpChainWithDuplicateAlias(t *testing.T) {
	// resolver mimics newHost: "user@a" and "a:22" both resolve to the
	// same host (EndPoint "a:22"), "other" is a distinct host
	var resolve func(jump string, chain []string) (*Host, error)
	resolve = func(jump string, chain []string) (*Host, error) {
		h := &Host{HostName: "a", Port: 22}
		if jump == "other" {
			h = &Host{HostName: "other", Port: 22}
		}
		var err error
		h.JumpList, err = buildJumpChainWith("", chain, resolve)
		return h, err
	}

	if _, err := buildJumpChainWith("user@a,a:22", nil, resolve); err == nil {
		t.Fatal("expected duplicate jump host error for alias spellings")
	}
	if _, err := buildJumpChainWith("a:22,other", nil, resolve); err != nil {
		t.Fatalf("distinct hosts should not be flagged: %v", err)
	}
}

func hostNames(hosts []*Host) []string {
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, h.HostName)
	}
	return names
}
