package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/logger"
	"github.com/PWZER/dssh/utils"
)

var (
	stdinReaderOnce sync.Once
	stdinReader     *bufio.Reader
)

// stdinBufioReader returns the process-wide shared reader for stdin so
// buffered piped input is not discarded between auth prompts and retries.
func stdinBufioReader() *bufio.Reader {
	stdinReaderOnce.Do(func() { stdinReader = bufio.NewReader(os.Stdin) })
	return stdinReader
}

// readAnswer reads one line from reader. TTY input is read with echo
// disabled via term.ReadPassword, piped input falls back to plain reads.
func readAnswer(reader *bufio.Reader, hidden bool) (string, error) {
	if hidden {
		byteAnswer, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", err
		}
		fmt.Println()
		return strings.TrimSpace(string(byteAnswer)), nil
	}
	line, err := reader.ReadString('\n')
	// keep the partial line when the input ends without a newline
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func getPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	return readAnswer(stdinBufioReader(), term.IsTerminal(int(syscall.Stdin)))
}

// doKeyboardInteractive answers keyboard-interactive questions. reader is
// shared across retries so buffered piped input is not discarded between
// attempts; hidden is computed once for the process.
func doKeyboardInteractive(reader *bufio.Reader, hidden bool) gossh.KeyboardInteractiveChallenge {
	return func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
		if instruction != "" {
			fmt.Println(instruction)
		}
		for i, question := range questions {
			fmt.Print(question)
			answer, err := readAnswer(reader, hidden && i < len(echos) && !echos[i])
			if err != nil {
				return answers, err
			}
			answers = append(answers, answer)
		}
		return answers, nil
	}
}

func getSignersCallback(host *config.Host) (signers []gossh.Signer, err error) {
	// prefer private keys already held by ssh-agent
	if a, err := NewAgent(); err != nil {
		logger.Warnf("ssh-agent error: %v", err)
	} else if agentSigners, err := a.Signers(); err != nil {
		logger.Warnf("ssh-agent signers error: %v", err)
	} else {
		signers = append(signers, agentSigners...)
	}

	// private key files
	for _, identityFile := range host.IdentityFiles {
		privateKeyBytes, err := os.ReadFile(identityFile)
		if err != nil {
			logger.Warnf("read private key file error: %v", err)
			continue
		}
		signer, err := gossh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			if _, ok := err.(*gossh.PassphraseMissingError); !ok {
				logger.Warnf("parse private key file %s error: %v", identityFile, err)
				continue
			}

			// ask for the private key passphrase
			prompt := fmt.Sprintf("[%s] Enter Identity Passphrase (%s)", host.Summary(), identityFile)
			password, err := getPassword(prompt)
			if err != nil {
				logger.Warnf("get password error: %v", err)
				continue
			}

			signer, err = gossh.ParsePrivateKeyWithPassphrase(privateKeyBytes, []byte(password))
			if err != nil {
				logger.Warnf("parse private key file %s with passphrase error: %v", identityFile, err)
				continue
			}
		}
		signers = append(signers, signer)
		logger.Debugf("use private key file: %s", identityFile)
	}
	return signers, nil
}

func CreateClientConfig(host *config.Host) *gossh.ClientConfig {
	var auth []gossh.AuthMethod

	// private keys
	auth = append(auth, gossh.PublicKeysCallback(func() (signers []gossh.Signer, err error) {
		return getSignersCallback(host)
	}))

	// fall back to typed password when key auth fails
	auth = append(auth, gossh.PasswordCallback(func() (string, error) {
		return getPassword(fmt.Sprintf("[%s] Enter Password: ", host.Summary()))
	}))

	// keyboard-interactive (2FA etc.), the shared stdin reader keeps
	// buffered piped input across retries
	auth = append(auth, gossh.RetryableAuthMethod(
		doKeyboardInteractive(stdinBufioReader(), term.IsTerminal(int(syscall.Stdin))),
		3,
	))

	return &gossh.ClientConfig{
		User:    host.Username,
		Auth:    auth,
		Timeout: utils.DialTimeout,
		BannerCallback: func(message string) error {
			fmt.Println(message)
			return nil
		},
		HostKeyCallback: HostKeyCallback,
	}
}
