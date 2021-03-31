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

func (c *Client) ForwardToAgent() (err error) {
	a, err := NewAgent()
	if err != nil {
		return err
	}
	return agent.ForwardToAgent(c.sshClient, a)
}

func (c *Client) MakeSession() (session *ssh.Session, finalize func(), err error) {
	if session, err = c.sshClient.NewSession(); err != nil {
		return session, finalize, err
	}

	finalize = func() {
		session.Close()
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {
		var oldState *terminal.State
		if oldState, err = terminal.MakeRaw(fd); err != nil {
			finalize()
			return session, finalize, err
		}

		finalize = func() {
			session.Close()
			terminal.Restore(fd, oldState)
		}

		var termWidth, termHeight int
		if termWidth, termHeight, err = terminal.GetSize(fd); err != nil {
			finalize()
			return session, finalize, err
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,     // enable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
	}
	// agent forward
	c.ForwardToAgent()
	agent.RequestAgentForwarding(session)
	// 自动调节窗口大小
	go c.UpdateTerminalSize(session)
	return session, finalize, err
}

func (c *Client) Execute(cmd string) (exitCode int, err error) {
	exitCode = -1
	session, finalize, err := c.MakeSession()
	if err != nil {
		return exitCode, err
	}
	defer finalize()
	if err = session.Start(cmd); err != nil {
		return exitCode, err
	}
	if err = session.Wait(); err != nil {
		if werr, ok := err.(*ssh.ExitError); ok {
			exitCode = werr.ExitStatus()
		}
	} else {
		exitCode = 0
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
	session, finalize, err := c.MakeSession()
	if err != nil {
		return err
	}
	defer finalize()

	if err = session.Shell(); err != nil {
		return err
	}
	session.Wait()
	return nil
}
