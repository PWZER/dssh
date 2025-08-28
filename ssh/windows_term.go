//go:build windows
// +build windows

package ssh

import (
	"golang.org/x/crypto/ssh"
)

// 监听窗口大小变化，并自动调节
func (c *Client) UpdateTerminalSize(session *ssh.Session) {
	// TODO
}
