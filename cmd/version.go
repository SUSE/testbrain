package cmd

import (
	"github.com/hpcloud/testbrain/lib"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display testbrain's version.",
	Run: func(cmd *cobra.Command, args []string) {
		lib.UI.Println(version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
