package ssh

import (
	"fmt"
	"strings"
	"syscall"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/PWZER/dssh/config"
)

func getPassword(host *config.Host) (password string, err error) {
	fmt.Printf("Enter Password (%s): ", host.String())
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return password, err
	}
	password = strings.TrimSpace(string(bytePassword))
	return password, err
}

func getKeyboardInteractive(host *config.Host, user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	if len(questions) > 0 {
		for idx := range questions {
			// echos[idx] == false
			var answer string
			fmt.Print(questions[idx])
			if _, err := fmt.Scan(&answer); err != nil {
				return answers, err
			}
			answers = append(answers, answer)
		}
	}
	return answers, nil
}

func GetClientConfig(host *config.Host) *gossh.ClientConfig {
	var auth []gossh.AuthMethod
	// 优先使用 ssh-agent 中已有的私钥
	if a, err := NewAgent(); err != nil {
		fmt.Println(err.Error())
	} else if signers, err := a.Signers(); err != nil {
		fmt.Println(err.Error())
	} else {
		auth = append(auth, gossh.PublicKeys(signers...))
	}
	// 私钥无法登录时，使用输入密码的方式
	qp := func() (string, error) { return getPassword(host) }
	auth = append(auth, gossh.PasswordCallback(qp))
	// 二次验证交互等
	qa := func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
		return getKeyboardInteractive(host, user, instruction, questions, echos)
	}
	auth = append(auth, gossh.KeyboardInteractive(qa))
	return &gossh.ClientConfig{
		User: host.User,
		Auth: auth,
		BannerCallback: func(message string) error {
			fmt.Println(message)
			return nil
		},
		HostKeyCallback:   gossh.InsecureIgnoreHostKey(),
		HostKeyAlgorithms: []string{gossh.KeyAlgoDSA, gossh.KeyAlgoRSA},
	}
}
