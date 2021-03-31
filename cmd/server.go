/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/PWZER/dssh/server"
	"github.com/PWZER/dssh/utils"
)

var fileServerConfig *server.FileServerConfig

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "simple file server",
	Long:  "simple file server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if fileServerConfig.Daemon {
			cmd, err := runAsDaemon()
			if err != nil || cmd != nil { // 异常/父进程应该退出
				return err
			}
		}
		return server.FileServerStart(fileServerConfig)
	},
}

func runAsDaemon() (cmd *exec.Cmd, err error) {
	envName := "DSSH_DAEMON_SUB_PROCESS"
	envValue := "yes"

	val := os.Getenv(envName)
	if val == envValue {
		return nil, nil
	}

	// 以下是父进程执行的代码
	var out *os.File
	if fileServerConfig.LogPath != "" {
		out, err = os.OpenFile(fileServerConfig.LogPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
	}

	cmd = exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", envName, envValue))
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, err
}

func init() {
	rootCmd.AddCommand(serverCmd)

	fileServerConfig = &server.FileServerConfig{}
	serverCmd.Flags().StringVarP(&fileServerConfig.Bind, "bind", "b", utils.GetLocalIP(), "bind host addr.")
	serverCmd.Flags().Int16VarP(&fileServerConfig.Port, "port", "p", 8000, "listen port.")
	serverCmd.Flags().StringVarP(&fileServerConfig.Root, "root", "r", "", "root path.")
	serverCmd.Flags().BoolVarP(&fileServerConfig.Daemon, "daemon", "d", false, "run as daemon.")
	serverCmd.Flags().StringVarP(&fileServerConfig.LogPath, "log_path", "", ".dssh_file_server.log", "log file path.")
}
