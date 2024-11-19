/*
Copyright © 2024 PWZER <pwzergo@gmail.com>
*/
package cmd

import (
	"github.com/PWZER/dssh/utils"
	"github.com/spf13/cobra"
)

var (
	dummy bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "upgrade dssh to the latest version",
	Long:  "upgrade dssh to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.Upgrade(dummy, Version)
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&dummy, "dummy", false, "dummy flag for testing")
}
