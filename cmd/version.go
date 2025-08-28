package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "v0.0.1"
var GitCommit = "<unknown>"

func printVersion() {
	fmt.Printf("dssh version: %s, git commit: %s\n", Version, GitCommit)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print dssh version",
	Long:  "print dssh version",
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
