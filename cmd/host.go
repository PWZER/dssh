/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

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
	"github.com/spf13/cobra"

	"github.com/PWZER/dssh/config"
)

var (
	hostName    string
	hostUser    string
	hostIP      string
	hostPort    uint16
	hostJump    string
	hostTags    string
	hostTimeout int = 0
)

// hostCmd represents the host command
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "host configs manage",
	Long:  "host configs manage",
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.ListConfigHosts(hostName, hostUser, hostTags)
	},
}

func addHostInit() {
	var addHostCmd = &cobra.Command{
		Use:   "add",
		Short: "add host",
		Long:  "add host",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.AddHost(hostName, hostUser, hostIP, hostPort, hostJump, hostTags, hostTimeout)
		},
	}

	addHostCmd.Flags().Bool("help", false, "help for this command.")
	addHostCmd.Flags().StringVarP(&hostName, "name", "n", "", "host name")
	addHostCmd.Flags().StringVarP(&hostUser, "user", "u", "", "login username")
	addHostCmd.Flags().StringVarP(&hostIP, "host", "h", "", "remote host ip")
	addHostCmd.Flags().Uint16VarP(&hostPort, "port", "p", 0, "remote host port")
	addHostCmd.Flags().StringVarP(&hostJump, "jump", "j", "", "proxy jump")
	addHostCmd.Flags().StringVarP(&hostTags, "tags", "t", "", "tags")
	addHostCmd.Flags().IntVarP(&hostTimeout, "timeout", "", 0, "timeout")
	addHostCmd.MarkFlagRequired("name")
	addHostCmd.MarkFlagRequired("host")
	hostCmd.AddCommand(addHostCmd)
}

func updateHostInit() {
	var updateHostCmd = &cobra.Command{
		Use:   "update",
		Short: "update host config",
		Long:  "update host config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.UpdateHost(hostName, hostUser, hostIP, hostPort, hostJump, hostTags, hostTimeout)
		},
	}

	updateHostCmd.Flags().Bool("help", false, "help for this command.")
	updateHostCmd.Flags().StringVarP(&hostName, "name", "n", "", "host name")
	updateHostCmd.Flags().StringVarP(&hostUser, "user", "u", "", "login username")
	updateHostCmd.Flags().StringVarP(&hostIP, "host", "h", "", "remote host ip")
	updateHostCmd.Flags().Uint16VarP(&hostPort, "port", "p", 0, "remote host port")
	updateHostCmd.Flags().StringVarP(&hostJump, "jump", "j", "", "proxy jump")
	updateHostCmd.Flags().StringVarP(&hostTags, "tags", "t", "", "tags")
	updateHostCmd.Flags().IntVarP(&hostTimeout, "timeout", "", -1, "timeout")
	updateHostCmd.MarkFlagRequired("name")
	hostCmd.AddCommand(updateHostCmd)
}

func delHostInit() {
	var delHostCmd = &cobra.Command{
		Use:   "del",
		Short: "del host",
		Long:  "del host",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.DeleteHost(hostName)
		},
	}
	delHostCmd.Flags().StringVarP(&hostName, "name", "n", "", "host name")
	delHostCmd.MarkFlagRequired("name")
	hostCmd.AddCommand(delHostCmd)
}

func init() {
	addHostInit()
	delHostInit()
	updateHostInit()

	hostCmd.Flags().Bool("help", false, "help for this command.")
	hostCmd.Flags().StringVarP(&hostName, "name", "n", "", "host name")
	hostCmd.Flags().StringVarP(&hostUser, "user", "u", "", "login username")
	hostCmd.Flags().StringVarP(&hostTags, "tags", "t", "", "tags")
	rootCmd.AddCommand(hostCmd)
}
