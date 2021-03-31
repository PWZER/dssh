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
	"github.com/spf13/cobra"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/PWZER/dssh/config"
	"github.com/PWZER/dssh/ssh"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           fmt.Sprintf("%s {host}...", os.Args[0]),
	Short:         "A command-line tools for ssh",
	Long:          "A command-line tools for ssh",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       "v1.0.0",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return ssh.Start(ssh.SSHConfig, args)
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

	// parallel
	rootCmd.Flags().IntVarP(&config.Config.Parallel, "parallel", "", 1, "max parallel run tasks num")

	// jump
	rootCmd.Flags().StringVarP(&config.Config.OverlayJump, "jump", "j", "", "ssh jump proxy")

	// user
	rootCmd.Flags().StringVarP(&config.Config.OverlayUser, "user", "u", "", "username")

	// user
	rootCmd.Flags().StringVar(&config.Config.OverlayHost, "host", "", "host name or remove host addr")

	// port
	rootCmd.Flags().Uint16VarP(&config.Config.OverlayPort, "port", "p", 0, "remote host port")

	// tags filter
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.Tags, "tags", "t", "", "tags filter")
	rootCmd.Flags().BoolVarP(&ssh.SSHConfig.FailedContinue, "force", "f", false, "force run when failed")

	// remote command
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.Command, "command", "c", "", "remote run command")
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.Script, "script", "s", "", "remote run script")
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.Module, "module", "m", "", "remote run module")

	// get
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.DownloadSrc, "get-src", "", "", "download remote src path")
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.DownloadDest, "get-dest", "", "", "download local dest path")

	// put
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.UploadSrc, "put-src", "", "", "upload local src path")
	rootCmd.Flags().StringVarP(&ssh.SSHConfig.UploadDest, "put-dest", "", "", "upload remote dest path")
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
		if err := config.InitConfig(); err != nil {
			fmt.Printf("load config file failed! %s err: %s", viper.ConfigFileUsed(), err.Error())
			os.Exit(1)
		}
	}
}
