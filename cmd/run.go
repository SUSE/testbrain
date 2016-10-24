package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hpcloud/testbrain/lib"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [flags] [files...]",
	Short: "Runs all tests.",
	Long: `Runs all bash tests given, gathering results and outputs
and summarizing it. If no files are given, the directory "test" is
assumed.  Any directories will be walked recursively.`,
	RunE: runCommandWithViperArgs,
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().Int("timeout", 300, "Timeout (in seconds) for each individual test")
	runCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	runCmd.PersistentFlags().BoolP("verbose", "v", false, "Output the progress of running tests")
	runCmd.PersistentFlags().String("include", "\\.sh$", "Regular expression of subset of tests to run")
	runCmd.PersistentFlags().String("exclude", "^$", "Regular expression of subset of tests to not run, applied after --include")

	viper.BindPFlags(runCmd.PersistentFlags())
}

type testPath struct {
	fullPath string
	relPath  string
}

func runCommandWithViperArgs(cmd *cobra.Command, args []string) error {
	timeoutInSeconds := viper.GetInt("timeout")
	flagJSONOutput := viper.GetBool("json")
	flagVerbose := viper.GetBool("verbose")
	flagTimeout := time.Duration(timeoutInSeconds) * time.Second
	flagInclude := viper.GetString("include")
	flagExclude := viper.GetString("exclude")
	if len(args) == 0 {
		// No args given, directory "test" is assumed
		args = []string{"test"}
	}
	return runCommand(args, flagInclude, flagExclude, flagTimeout, flagJSONOutput, flagVerbose)
}

func runCommand(testFolders []string, includeRe, excludeRe string, timeout time.Duration, jsonOutput bool, verbose bool) error {
	testFiles, err := getTestScripts(testFolders, includeRe, excludeRe)
	if err != nil {
		return err
	}
	if !jsonOutput {
		ui.Printf("Found %d test files\n", len(testFiles))
	}

	outputIndividualResults := !jsonOutput && verbose
	testResults := runAllTests(testFiles, timeout, outputIndividualResults)

	failedTestResults := getFailedTestResults(testResults)
	if jsonOutput {
		outputResultsJSON(failedTestResults, len(testResults))
	} else {
		outputResults(failedTestResults, len(testResults))
	}
	if len(failedTestResults) == 0 {
		return nil
	}
	return errors.New("Some tests failed")
}

func getTestScripts(testFolders []string, include, exclude string) ([]testPath, error) {
	includeRe, err := regexp.Compile(include)
	if err != nil {
		return nil, fmt.Errorf("Error parsing files to include: %s", err)
	}
	excludeRe, err := regexp.Compile(exclude)
	if err != nil {
		return nil, fmt.Errorf("Error parsing files to exclude: %s", err)
	}

	var foundTests []testPath
	for _, testFolder := range testFolders {
		info, err := os.Stat(testFolder)
		if err != nil {
			return nil, fmt.Errorf("Error reading test file %s: %s", testFolder, err)
		}
		if !info.IsDir() {
			// This is an individual test
			foundTests = append(foundTests, testPath{fullPath: testFolder, relPath: testFolder})
			continue
		}
		err = filepath.Walk(testFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !includeRe.MatchString(path) {
				return nil
			}
			if excludeRe.MatchString(path) {
				return nil
			}
			relPath, err := filepath.Rel(testFolder, path)
			if err != nil {
				return err
			}
			foundTests = append(foundTests, testPath{fullPath: path, relPath: relPath})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return foundTests, nil
}

func runAllTests(testFiles []testPath, timeout time.Duration, outputIndividualResults bool) []lib.TestResult {
	var testResults []lib.TestResult
	for i, testFile := range testFiles {
		if outputIndividualResults {
			ui.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := runSingleTest(testFile, timeout)
		printVerboseSingleTestResult(result, outputIndividualResults)
		testResults = append(testResults, result)
	}
	return testResults
}

func runSingleTest(testFile testPath, timeout time.Duration) lib.TestResult {
	command := exec.Command(testFile.fullPath)
	commandOutput := &bytes.Buffer{}
	command.Stdout = commandOutput
	command.Stderr = commandOutput
	err := command.Start()
	if err != nil {
		return lib.ErrorTestResult(testFile.relPath, err)
	}
	timer := time.AfterFunc(timeout, func() {
		command.Process.Kill()
		commandOutput.WriteString(fmt.Sprintf("Killed by testbrain: Timed out after %v", timeout))
	})
	err = command.Wait()
	timer.Stop()

	exitCode, err := getErrorCode(err, command)
	if err != nil {
		return lib.ErrorTestResult(testFile.relPath, err)
	}

	return lib.TestResult{
		TestFile: testFile.relPath,
		Success:  command.ProcessState.Success(),
		ExitCode: exitCode,
		Output:   string(commandOutput.Bytes()),
	}
}

func getErrorCode(err error, command *exec.Cmd) (int, error) {
	if command.ProcessState.Success() {
		// Not exactly necessary, since we can check Success(),
		// but more correct than saying status code is unknown
		return 0, nil
	}
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			// The program has exited with an ExitError, we can get the error code from WaitStatus
			// See https://golang.org/pkg/os/#ProcessState.Sys
			// Although the docs mention syscall.WaitStatus works on Unix, it seems to work on Windows too
			status, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				return status.ExitStatus(), nil
			}
		}

		// There is an error but it's not an ExitError
		// Something other than the test script failed, bubble up the error
		return lib.UnknownExitCode, err
	}

	// The test script failed, but without an error
	return lib.UnknownExitCode, nil
}

func printVerboseSingleTestResult(result lib.TestResult, outputIndividualResults bool) {
	if outputIndividualResults {
		if result.Success {
			green.Println("OK")
		} else {
			redBold.Println("FAILED")
			red.Printf("Output:\n%s\n", result.Output)
		}
	}
}

func getFailedTestResults(testResults []lib.TestResult) []lib.TestResult {
	failedTestResults := []lib.TestResult{}
	for _, result := range testResults {
		if !result.Success {
			failedTestResults = append(failedTestResults, result)
		}
	}
	return failedTestResults
}

func outputResults(failedTestResults []lib.TestResult, nbTestsRan int) {
	for _, failedResult := range failedTestResults {
		redBold.Printf("%s: Failed with exit code %d\n", failedResult.TestFile, failedResult.ExitCode)
	}
	nbTestsPassed := nbTestsRan - len(failedTestResults)
	summaryString := fmt.Sprintf("\nTests complete: %d Passed, %d Failed", nbTestsPassed, len(failedTestResults))
	if len(failedTestResults) > 0 {
		redBold.Println(summaryString)
	} else {
		greenBold.Println(summaryString)
	}
}

func outputResultsJSON(failedTestResults []lib.TestResult, nbTestsRan int) {
	// This is the only place where we need this struct, so anonymous struct seems appropriate
	jsonOutputStruct := struct {
		Passed     int              `json:"passed"`
		Failed     int              `json:"failed"`
		FailedList []lib.TestResult `json:"failedList"`
	}{
		Passed:     nbTestsRan - len(failedTestResults),
		Failed:     len(failedTestResults),
		FailedList: failedTestResults,
	}
	jsonOutput, _ := json.Marshal(jsonOutputStruct)
	ui.Write(jsonOutput)
}
