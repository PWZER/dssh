package ssh

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/logger"
	"github.com/PWZER/dssh/utils"
)

type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func NewClient() *Client {
	return &Client{}
}

// bound the handshake over the tunnel: chanConn does not support
// deadlines, close the connection when the handshake hangs
func (c *Client) Connect(host *config.Host) (err error) {
	clientConfig := CreateClientConfig(host)
	if c.sshClient == nil {
		c.sshClient, err = ssh.Dial("tcp", host.EndPoint(), clientConfig)
		return err
	}
	dial, err := c.sshClient.Dial("tcp", host.EndPoint())
	if err != nil {
		return err
	}
	timer := time.AfterFunc(utils.DialTimeout, func() { dial.Close() })
	conn, chans, reqs, err := ssh.NewClientConn(dial, host.EndPoint(), clientConfig)
	if !timer.Stop() {
		// timer already fired, the connection was closed by the callback
		if err == nil {
			conn.Close()
			return fmt.Errorf("handshake timeout for %s", host.EndPoint())
		}
		dial.Close()
		return err
	}
	if err != nil {
		dial.Close()
		return err
	}
	c.sshClient = ssh.NewClient(conn, chans, reqs)
	return nil
}

func (c *Client) RequestAgentForwarding(session *ssh.Session) error {
	a, err := NewAgent()
	if err != nil {
		return err
	}
	if err := agent.ForwardToAgent(c.sshClient, a); err != nil {
		return err
	}
	return agent.RequestAgentForwarding(session)
}

func (c *Client) MakeSession() (*ssh.Session, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return session, err
	}
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	return session, nil
}

func (c *Client) Execute(cmd string) (int, error) {
	exitCode := 0
	session, err := c.MakeSession()
	if err != nil {
		return exitCode, err
	}
	defer session.Close()

	if err = session.Start(cmd); err != nil {
		return exitCode, err
	}
	if err = session.Wait(); err != nil {
		if werr, ok := err.(*ssh.ExitError); ok {
			exitCode = werr.ExitStatus()
		}
	}
	return exitCode, err
}

func (c *Client) Script(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return -1, err
	}
	return c.Execute(string(content))
}

func (c *Client) Shell(remoteListen, proxyServer string) error {
	session, err := c.MakeSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// agent forward
	if err := c.RequestAgentForwarding(session); err != nil {
		logger.Warnf("agent forwarding disabled: %v", err)
	}

	// remote proxy
	if remoteListen != "" && proxyServer != "" {
		go func() {
			if err := c.RemoteProxy(remoteListen, proxyServer); err != nil {
				logger.Errorf("remote proxy: %v", err)
			}
		}()
	}

	// auto update window size
	go c.UpdateTerminalSize(session)

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer term.Restore(fd, oldState)

		termWidth, termHeight, err := term.GetSize(fd)
		if err != nil {
			return err
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,     // enable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
			return err
		}
	}

	if err = session.Shell(); err != nil {
		return err
	}
	return session.Wait()
}

func (c *Client) RemoteProxy(listenAddr, serverAddr string) error {
	listener, err := c.sshClient.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %s error: %w", listenAddr, err)
	}
	defer listener.Close()
	logger.Infof("Listening on %s", listenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept error: %w", err)
		}
		logger.Infof("Accepted from %s", conn.RemoteAddr())
		go func(conn net.Conn) {
			defer conn.Close()
			if err := utils.CopyConn(conn, serverAddr); err != nil {
				logger.Errorf("CopyConn Error: %v", err)
			}
		}(conn)
	}
}
