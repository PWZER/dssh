package ssh

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/utils"
	"golang.org/x/term"
)

func taskStart(task *config.Task) (err error) {
	client := NewClient()
	for _, host := range append(task.Target.JumpList, task.Target) {
		if err = client.Connect(host); err != nil {
			return err
		}
	}

	if task.Command != "" {
		_, err = client.Execute(task.Command)
		return err
	}

	if task.DownloadSrc != "" {
		return client.Download(task.DownloadSrc, task.DownloadDest)
	}

	if task.UploadSrc != "" {
		return client.Upload(task.UploadSrc, task.UploadDest)
	}

	utils.SetWindowTitle(task.Target.HostName)
	defer utils.SetWindowTitle("")
	return client.Shell(task.RemoteListen, task.ProxyServer)
}

func Start(tc *config.TaskConfig) error {
	if len(tc.Tasks) == 0 {
		return fmt.Errorf("one of \"<host>\" or \"--host <host>\" or \"--tags\" is required!")
	}

	for _, task := range tc.Tasks {
		termWidth, _, err := term.GetSize(int(os.Stdin.Fd()))
		if err == nil && termWidth > 0 {
			message := fmt.Sprintf("-----> [%d / %d] %s %s <-----",
				task.Index+1, len(tc.Tasks), task.Target.Summary(), task.Message)
			fillLen := termWidth - int(math.Mod(float64(len(message)), float64(termWidth)))
			if fillLen > 0 {
				message = fmt.Sprintf("\033[1;32m%s%s\033[0m", message, strings.Repeat("-", fillLen))
			}
			fmt.Fprintln(os.Stderr, message)
		}

		if err := taskStart(task); err != nil {
			if tc.FailedContinue {
				fmt.Printf("[ERROR] %s\n", err)
				continue
			}
			return err
		}
	}
	return nil
}
