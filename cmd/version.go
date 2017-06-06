package cmd

import (
	"github.com/spf13/cobra"
	"github.com/suse/testbrain/lib"
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
