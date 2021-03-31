package ssh

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/utils"
)

type Task struct {
	Index        int
	Target       *config.Host
	Command      string
	Message      string
	Outputer     string
	UploadSrc    string
	UploadDest   string
	DownloadSrc  string
	DownloadDest string
}

func (task *Task) ParseCommand(command, script, module string) error {
	jump := task.Target.JumpString()
	if jump != "" {
		task.Message = fmt.Sprintf("jump: %s", jump)
	}
	if command == "" {
		if script == "" && module != "" {
			script = filepath.Join(config.Config.ModulesDir, module+".sh")
		}

		if script != "" {
			content, err := ioutil.ReadFile(script)
			if err != nil {
				return err
			}
			task.Command = string(content)
			task.Message = fmt.Sprintf("%s script: %s ", task.Message, script)
		}
	} else {
		task.Command = command
		task.Message = fmt.Sprintf("%s command: %s ", task.Message, command)
	}
	return nil
}

func (task *Task) Start() (err error) {
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

	utils.SetWindowTitle(task.Target.Name)
	defer utils.SetWindowTitle("")
	return client.Shell()
}
