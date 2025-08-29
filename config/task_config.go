package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Task struct {
	Index        int
	Target       *Host
	Command      string
	RemoteListen string
	ProxyServer  string
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
			script = filepath.Join(Config.ModulesDir, module+".sh")
		}

		if script != "" {
			content, err := os.ReadFile(script)
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

type TaskConfig struct {
	Username       string
	Port           uint16
	ProxyJump      string
	IdentityFiles  []string
	Tags           []string
	Targets        []string
	RemoteListen   string
	ProxyServer    string
	Command        string
	Script         string
	Module         string
	UploadSrc      string
	UploadDest     string
	DownloadSrc    string
	DownloadDest   string
	FailedContinue bool
	Parallel       int
	Tasks          []*Task
}

func NewTaskConfig() *TaskConfig {
	return &TaskConfig{
		Username:       "",
		Port:           0,
		IdentityFiles:  []string{},
		Parallel:       1,
		FailedContinue: false,
	}
}

func (cfg *TaskConfig) addTask(target string) (err error) {
	host, err := NewHost(cfg.Username, target, cfg.Port, cfg.ProxyJump, cfg.IdentityFiles)
	if err != nil {
		return err
	}

	task := &Task{
		Index:        len(cfg.Tasks),
		Target:       host,
		RemoteListen: cfg.RemoteListen,
		ProxyServer:  cfg.ProxyServer,
		UploadSrc:    cfg.UploadSrc,
		UploadDest:   cfg.UploadDest,
		DownloadSrc:  cfg.DownloadSrc,
		DownloadDest: cfg.DownloadDest,
	}
	if err = task.ParseCommand(cfg.Command, cfg.Script, cfg.Module); err != nil {
		return err
	}
	cfg.Tasks = append(cfg.Tasks, task)
	return nil
}

func (cfg *TaskConfig) InitTasks() error {
	if len(cfg.Tags) > 0 {
		for _, tag := range cfg.Tags {
			if matched, err := regexp.MatchString(`[0-9a-zA-z_\-,]*`, tag); err != nil || !matched {
				return fmt.Errorf("invalid tags: %s", tag)
			}
		}

		hosts, err := GetHostsFromSSHConfig()
		if err != nil {
			return err
		}

		for _, host := range hosts {
			if !host.MatchTags(cfg.Tags) {
				continue
			}
			task := &Task{
				Index:        len(cfg.Tasks),
				Target:       host,
				RemoteListen: cfg.RemoteListen,
				ProxyServer:  cfg.ProxyServer,
				UploadSrc:    cfg.UploadSrc,
				UploadDest:   cfg.UploadDest,
				DownloadSrc:  cfg.DownloadSrc,
				DownloadDest: cfg.DownloadDest,
			}
			if err = task.ParseCommand(cfg.Command, cfg.Script, cfg.Module); err != nil {
				return err
			}
			cfg.Tasks = append(cfg.Tasks, task)
		}
	} else {
		for _, target := range cfg.Targets {
			for _, hostString := range strings.Split(target, ",") {
				if err := cfg.addTask(hostString); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
