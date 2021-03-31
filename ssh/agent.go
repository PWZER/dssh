package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh/agent"

	"github.com/PWZER/dssh/config"
)

func NewAgent() (agent.Agent, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("SSH_AUTH_SOCK environment variable is not set!")
	}
	agent_conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}
	return agent.NewClient(agent_conn), nil
}

func FixSSHAuth() error {
	files, err := filepath.Glob("/tmp/ssh-*/agent*")
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("not found ssh agent forward!")
		return nil
	}

	os.Remove(config.Config.SSHAuthSock)

	var lastSSHAuthSock string
	var lastSSHAuthSockUpdateTime time.Time = time.Time{}
	for _, filename := range files {
		if err := os.Chown(filename, os.Getuid(), os.Getgid()); err != nil {
			continue
		}
		stat, err := os.Stat(filename)
		if err != nil {
			continue
		}
		if stat.ModTime().After(lastSSHAuthSockUpdateTime) {
			lastSSHAuthSock = filename
			lastSSHAuthSockUpdateTime = stat.ModTime()
		}
	}
	if err := os.Symlink(lastSSHAuthSock, config.Config.SSHAuthSock); err != nil {
		return err
	}
	return nil
}
