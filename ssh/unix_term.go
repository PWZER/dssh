// +build linux darwin

package ssh

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// 监听窗口大小变化，并自动调节
func (c *Client) UpdateTerminalSize(session *ssh.Session) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGWINCH)

	var termWidth, termHeight int
	for receivedSignal := range signalChan {
		if receivedSignal == nil {
			break
		}
		if currTermWidth, currTermHeight, err := terminal.GetSize(int(os.Stdin.Fd())); err != nil {
			continue
		} else if currTermHeight != termHeight || currTermWidth != termWidth {
			if err := session.WindowChange(currTermHeight, currTermWidth); err != nil {
				continue
			}
			termWidth, termHeight = currTermWidth, currTermHeight
		}
	}
}
