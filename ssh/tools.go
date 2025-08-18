package ssh

import (
	"os"
	"strings"

	"golang.org/x/term"
)

func GetSSHLocalHost() string {
	// ssh login server
	sshConnInfo := strings.Split(os.Getenv("SSH_CONNECTION"), " ")
	if len(sshConnInfo) == 4 && len(sshConnInfo[2]) > 0 {
		return sshConnInfo[2]
	}
	return "127.0.0.1"
}

func GetTerminalWidth() int {
	termWidth, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 80
	}
	return termWidth
}
