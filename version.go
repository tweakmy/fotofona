package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Use for build version
var buildtimestamp string

// Use to indicate which fotofona version
var githash string

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Fotofona",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(`Fotofona build:
	TimeStamp: %s
	GitHash: %s
`, buildtimestamp, githash)
	},
}
