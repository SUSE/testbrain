package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hpcloud/termui"
	"github.com/hpcloud/termui/termpassword"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hpcloud/testbrain/lib"
)

var cfgFile string
var version string

// Creating printers globally to simplify printing
var (
	ui        *termui.UI
	green     = color.New(color.FgGreen)
	greenBold = color.New(color.FgGreen, color.Bold)
	red       = color.New(color.FgRed)
	redBold   = color.New(color.FgRed, color.Bold)
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "testbrain",
	Short: "Acceptance tests brain for HCF",
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(versionArg string) {
	version = versionArg
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.test-brain.yaml)")
	ui = termui.New(os.Stdin, lib.Writer, termpassword.NewReader())
	// This lets us use the standard Print functions of the color library while printing to the UI
	color.Output = ui
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".test-brain") // name of config file (without extension)
	viper.AddConfigPath("$HOME")       // adding home directory as first search path
	viper.SetEnvPrefix("TESTBRAIN")    // all env vars will start with TESTBRAIN_
	viper.AutomaticEnv()               // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
