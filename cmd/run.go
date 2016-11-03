package cmd

import (
	"errors"
	"os"
	"time"

	"github.com/hpcloud/testbrain/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [flags] [files...]",
	Short: "Runs all tests.",
	Long: `Runs all bash tests given, gathering results and outputs
and summarizing it. If no files are given, the current
working directory is assumed.  Any directories will be
walked recursively.`,
	RunE: runCommandWithViperArgs,
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().Int("timeout", 300, "Timeout (in seconds) for each individual test")
	runCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	runCmd.PersistentFlags().BoolP("verbose", "v", false, "Output the progress of running tests")
	runCmd.PersistentFlags().String("include", "_test\\.sh$", "Regular expression of subset of tests to run")
	runCmd.PersistentFlags().String("exclude", "^$", "Regular expression of subset of tests to not run, applied after --include")
	runCmd.PersistentFlags().Bool("in-order", false, "Do not randomize test order")
	runCmd.PersistentFlags().Int64("seed", -1, "Random seed used to determine the order of tests")
	runCmd.PersistentFlags().BoolP("dry-run", "n", false, "Do not actually run the tests")

	viper.BindPFlags(runCmd.PersistentFlags())
}

func runCommandWithViperArgs(_ *cobra.Command, args []string) error {
	timeoutInSeconds := viper.GetInt("timeout")
	flagTimeout := time.Duration(timeoutInSeconds) * time.Second
	flagJSONOutput := viper.GetBool("json")
	flagVerbose := viper.GetBool("verbose")
	flagInclude := viper.GetString("include")
	flagExclude := viper.GetString("exclude")
	flagInOrder := viper.GetBool("in-order")
	flagSeed := viper.GetInt64("seed")
	flagDryRun := viper.GetBool("dry-run")

	if flagInOrder && flagSeed != -1 {
		return errors.New("Cannot set --in-order and --seed at the same time")
	}
	if flagSeed == -1 {
		flagSeed = time.Now().UnixNano()
	}

	if len(args) == 0 {
		// No args given, current working directory is assumed
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = []string{cwd}
	}

	runner := &lib.Runner{
		TestTargets: args,
		IncludeRe:   flagInclude,
		ExcludeRe:   flagExclude,
		Timeout:     flagTimeout,
		InOrder:     flagInOrder,
		RandomSeed:  flagSeed,
		JSONOutput:  flagJSONOutput,
		Verbose:     flagVerbose,
		DryRun:      flagDryRun,
	}
	return runner.RunCommand()
}
