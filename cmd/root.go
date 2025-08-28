/*
Copyright © 2020 PWZER <pwzergo@gmail.com>

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

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/logger"
	"github.com/PWZER/dssh/ssh"
)

var cfgFile string
var showVersion bool
var taskConfig *config.TaskConfig = config.NewTaskConfig()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           fmt.Sprintf("%s {host}...", os.Args[0]),
	Short:         "A command-line tools for ssh",
	Long:          "A command-line tools for ssh",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			printVersion()
			return nil
		}

		if len(taskConfig.Targets) > 0 {
			if len(args) > 0 {
				return fmt.Errorf("host name and args can not be used together")
			}
		} else {
			if len(args) == 0 {
				return fmt.Errorf("host name is required")
			}
			taskConfig.Targets = append(taskConfig.Targets, args...)
		}
		if err := taskConfig.InitTasks(); err != nil {
			return err
		}
		return ssh.Start(taskConfig)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return config.GetHostNames(), cobra.ShellCompDirectiveNoFileComp
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("[ERROR]", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.dssh.yaml)")
	rootCmd.PersistentFlags().VarP(&logger.LogLevel, "log-level", "l", "log level, allowed ( debug, info, warn, error, fatal, panic )")

	// version
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version")

	// task config
	rootCmd.Flags().StringArrayVar(&taskConfig.Targets, "host", []string{}, "host name")
	rootCmd.Flags().StringVarP(&taskConfig.Username, "user", "u", "", "username")
	rootCmd.Flags().Uint16VarP(&taskConfig.Port, "port", "p", 0, "remote host port")
	rootCmd.Flags().StringArrayVar(&taskConfig.IdentityFiles, "identity", []string{}, "identity file")
	rootCmd.Flags().IntVarP(&taskConfig.Parallel, "parallel", "", 1, "max parallel run tasks num")
	rootCmd.Flags().StringVarP(&taskConfig.Tags, "tags", "t", "", "tags filter")
	rootCmd.Flags().BoolVarP(&taskConfig.FailedContinue, "force", "f", false, "force run when failed")

	// remote command
	rootCmd.Flags().StringVarP(&taskConfig.Command, "command", "c", "", "remote run command")
	rootCmd.Flags().StringVarP(&taskConfig.Script, "script", "s", "", "remote run script")
	rootCmd.Flags().StringVarP(&taskConfig.Module, "module", "m", "", "remote run module")

	// remote proxy
	rootCmd.Flags().StringVar(&taskConfig.RemoteListen, "remote-listen", "", "remote proxy listen address")
	rootCmd.Flags().StringVar(&taskConfig.ProxyServer, "proxy-server", "", "proxy server address")

	// get
	rootCmd.Flags().StringVarP(&taskConfig.DownloadSrc, "get-src", "", "", "download remote src path")
	rootCmd.Flags().StringVarP(&taskConfig.DownloadDest, "get-dest", "", "", "download local dest path")

	// put
	rootCmd.Flags().StringVarP(&taskConfig.UploadSrc, "put-src", "", "", "upload local src path")
	rootCmd.Flags().StringVarP(&taskConfig.UploadDest, "put-dest", "", "", "upload remote dest path")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".dssh")
	}

	viper.AutomaticEnv()
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err == nil {
		if err := config.LoadConfig(); err != nil {
			fmt.Printf("load config file failed! %s err: %s", viper.ConfigFileUsed(), err.Error())
			os.Exit(1)
		}
	}
}
