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
	"github.com/spf13/cobra"

	"github.com/PWZER/dssh/ssh"
)

// fixCmd represents the fix command
var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "fix ssh agent forward",
	Long:  "fix ssh agent forward",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ssh.FixSSHAuth()
	},
}

func init() {
	rootCmd.AddCommand(fixCmd)
}
