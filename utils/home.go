package utils

import (
	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"
)

// HomeDir returns the current user's home directory.
func HomeDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return dir
}

// DialTimeout bounds the TCP dial and the SSH handshake of every hop.
const DialTimeout = 30 * time.Second
