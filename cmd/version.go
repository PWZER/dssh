package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "v0.0.1"
var GitCommit = "<unknown>"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print dssh version",
	Long:  "print dssh version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dssh version: %s, git commit: %s\n", Version, GitCommit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
