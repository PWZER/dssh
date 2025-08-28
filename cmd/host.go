/*
Copyright © 2021 PWZER <pwzergo@gmail.com>

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
	hostName string
	hostUser string
	hostTags string
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

func init() {
	hostCmd.Flags().Bool("help", false, "help for this command.")
	hostCmd.Flags().StringVarP(&hostName, "name", "n", "", "host name")
	hostCmd.Flags().StringVarP(&hostUser, "user", "u", "", "login username")
	hostCmd.Flags().StringVarP(&hostTags, "tags", "t", "", "tags")
	rootCmd.AddCommand(hostCmd)
}
