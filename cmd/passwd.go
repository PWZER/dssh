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

	"github.com/PWZER/dssh/utils"
)

var pg *utils.PasswordGenerator

// passwdCmd represents the passwd command
var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "password generator",
	Long:  "password generator",
	RunE: func(cmd *cobra.Command, args []string) error {
		return pg.GenPassword()
	},
}

func init() {
	rootCmd.AddCommand(passwdCmd)

	pg = &utils.PasswordGenerator{}

	passwdCmd.Flags().BoolVarP(&pg.DisabledDigital, "disabledDigital", "D", false, "disabled Digital")
	passwdCmd.Flags().BoolVarP(&pg.DisabledLowercase, "disabledLowercase", "L", false, "disabled Lowercase")
	passwdCmd.Flags().BoolVarP(&pg.DisabledUppercase, "disabledUppercase", "U", false, "disabled Uppercase")
	passwdCmd.Flags().BoolVarP(&pg.DisabledPunctuation, "disabledPunctuation", "P", false, "disabled Punctuation")
	passwdCmd.Flags().IntVarP(&pg.PasswordLength, "length", "l", 16, "password length")
}
