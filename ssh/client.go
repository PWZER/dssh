package ssh

import (
	"io/ioutil"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/PWZER/dssh/config"
)

type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect(host *config.Host) (err error) {
	config := GetClientConfig(host)
	if c.sshClient == nil {
		c.sshClient, err = ssh.Dial("tcp", host.EndPoint(), config)
		return err
	}
	dial, err := c.sshClient.Dial("tcp", host.EndPoint())
	if err != nil {
		return err
	}
	conn, chans, reqs, err := ssh.NewClientConn(dial, host.EndPoint(), config)
	if err != nil {
		return err
	}
	c.sshClient = ssh.NewClient(conn, chans, reqs)
	return err
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
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}
	return c.Execute(string(content))
}

func (c *Client) Shell() error {
	session, err := c.MakeSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// agent forward
	if err := c.RequestAgentForwarding(session); err != nil {
		return err
	}

	// auto update window size
	go c.UpdateTerminalSize(session)

	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		oldState, err := terminal.MakeRaw(fd)
		if err != nil {
			return err
		}
		defer terminal.Restore(fd, oldState)

		termWidth, termHeight, err := terminal.GetSize(fd)
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
