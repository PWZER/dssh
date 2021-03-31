package ssh

import (
	"fmt"
	"math"
	"strings"

	"github.com/PWZER/dssh/config"
)

type SSHConfigType struct {
	Tasks          []*Task
	Tags           string
	Command        string
	Script         string
	Module         string
	UploadSrc      string
	UploadDest     string
	DownloadSrc    string
	DownloadDest   string
	FailedContinue bool
}

var SSHConfig = &SSHConfigType{}

func (cfg *SSHConfigType) matchHost(hostString string) (host *config.Host, err error) {
	if host, ok := config.Config.Hosts[hostString]; ok {
		return host, err
	}
	host = &config.Host{Addr: hostString}
	err = host.Parse()
	return host, err
}

func (cfg *SSHConfigType) parseTask(name string) (hasMatch bool, err error) {
	host, err := cfg.matchHost(name)
	if err != nil {
		return hasMatch, err
	}

	task := &Task{
		Index: len(cfg.Tasks), Target: host,
		UploadSrc: cfg.UploadSrc, UploadDest: cfg.UploadDest,
		DownloadSrc: cfg.DownloadSrc, DownloadDest: cfg.DownloadDest,
	}
	if err = task.ParseCommand(cfg.Command, cfg.Script, cfg.Module); err != nil {
		return hasMatch, err
	}
	cfg.Tasks = append(cfg.Tasks, task)
	return true, err
}

func (cfg *SSHConfigType) addRemoteTarget(hostString string) error {
	if hasMatch, err := cfg.parseTask(hostString); err != nil || hasMatch {
		return err
	}

	host := &config.Host{Addr: hostString}
	if err := host.Parse(); err != nil {
		return err
	}

	task := &Task{
		Index: len(cfg.Tasks), Target: host,
		UploadSrc: cfg.UploadSrc, UploadDest: cfg.UploadDest,
		DownloadSrc: cfg.DownloadSrc, DownloadDest: cfg.DownloadDest,
	}
	if err := task.ParseCommand(cfg.Command, cfg.Script, cfg.Module); err != nil {
		return err
	}
	cfg.Tasks = append(cfg.Tasks, task)
	return nil
}

func (cfg *SSHConfigType) initTasks(targets []string) error {
	if cfg.Tags != "" {
		if err := config.CheckTags(cfg.Tags); err != nil {
			return err
		}
		hosts, err := config.ConfigHostsFilter("", "", cfg.Tags)
		if err != nil {
			return err
		}

		for _, host := range hosts {
			task := &Task{
				Index: len(cfg.Tasks), Target: host,
				UploadSrc: cfg.UploadSrc, UploadDest: cfg.UploadDest,
				DownloadSrc: cfg.DownloadSrc, DownloadDest: cfg.DownloadDest,
			}
			if err = task.ParseCommand(cfg.Command, cfg.Script, cfg.Module); err != nil {
				return err
			}
			cfg.Tasks = append(cfg.Tasks, task)
		}
	} else {
		for _, target := range targets {
			for _, hostString := range strings.Split(target, ",") {
				if err := cfg.addRemoteTarget(hostString); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (cfg *SSHConfigType) start(targets []string) error {
	if err := cfg.initTasks(targets); err != nil {
		return err
	}

	if len(cfg.Tasks) == 0 {
		return fmt.Errorf("one of \"<host>\" or \"--host <host>\" or \"--tags\" is required!")
	}

	for _, task := range cfg.Tasks {
		task.Target.Overlay()
		message := fmt.Sprintf("-----> [%d / %d] %s %s <-----",
			task.Index+1, len(cfg.Tasks), task.Target.String(), task.Message)
		terminalWidth := GetTerminalWidth()
		fillLen := terminalWidth - int(math.Mod(float64(len(message)), float64(terminalWidth)))
		if fillLen > 0 {
			message = fmt.Sprintf("\033[1;32m%s%s\033[0m", message, strings.Repeat("-", fillLen))
		}
		fmt.Println(message)

		if err := task.Start(); err != nil {
			if cfg.FailedContinue {
				fmt.Printf("[ERROR] %s\n", err)
				continue
			}
			return err
		}
	}
	return nil
}

func Start(cfg *SSHConfigType, targets []string) error {
	if config.Config.OverlayHost != "" {
		targets = append(targets, config.Config.OverlayHost)
	}
	return cfg.start(targets)
}
