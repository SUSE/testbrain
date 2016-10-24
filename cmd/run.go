package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/hpcloud/termui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hpcloud/testbrain/lib"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs all tests.",
	Long: `Runs all bash tests in the designated test folder,
gathering results and outputs and summarizing it.`,
	RunE: runCommandWithViperArgs,
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().String("testfolder", "tests", "Folder containing the test files to run")
	runCmd.PersistentFlags().Int("timeout", 300, "Timeout (in seconds) for each individual test")
	runCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	runCmd.PersistentFlags().BoolP("verbose", "v", false, "Output the progress of running tests")

	viper.BindPFlags(runCmd.PersistentFlags())
}

func runCommandWithViperArgs(cmd *cobra.Command, args []string) error {
	flagTestFolder := viper.GetString("testfolder")
	timeoutInSeconds := viper.GetInt("timeout")
	flagJSONOutput := viper.GetBool("json")
	flagVerbose := viper.GetBool("verbose")
	flagTimeout := time.Duration(timeoutInSeconds) * time.Second
	return runCommand(flagTestFolder, flagTimeout, flagJSONOutput, flagVerbose)
}

func runCommand(testFolder string, timeout time.Duration, jsonOutput bool, verbose bool) error {
	testFiles := getTestScripts(testFolder)
	if !jsonOutput {
		ui.Printf("Found %d test files\n", len(testFiles))
	}

	outputIndividualResults := !jsonOutput && verbose
	testResults := runAllTests(testFiles, testFolder, timeout, outputIndividualResults)

	failedTestResults := getFailedTestResults(testResults)
	if jsonOutput {
		outputResultsJSON(failedTestResults, len(testResults))
	} else {
		outputResults(failedTestResults, len(testResults))
	}
	if len(failedTestResults) == 0 {
		return nil
	}
	return fmt.Errorf("%d tests failed", len(failedTestResults))
}

func getTestScripts(testFolder string) []string {
	fileList, err := ioutil.ReadDir(testFolder)
	if err != nil {
		redBold.Println("Could not open test folder: " + testFolder)
		termui.PrintAndExit(ui, err)
	}
	var testFileList []string
	for _, file := range fileList {
		if strings.HasSuffix(file.Name(), ".sh") {
			testFileList = append(testFileList, file.Name())
		}
	}
	return testFileList
}

func runAllTests(testFiles []string, testFolder string,
	timeout time.Duration, outputIndividualResults bool) []lib.TestResult {
	var testResults []lib.TestResult
	for i, testFile := range testFiles {
		if outputIndividualResults {
			ui.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := runSingleTest(testFile, testFolder, timeout)
		printVerboseSingleTestResult(result, outputIndividualResults)
		testResults = append(testResults, result)
	}
	return testResults
}

func runSingleTest(testFile string, testFolder string, timeout time.Duration) lib.TestResult {
	command := exec.Command(path.Join(testFolder, testFile))
	commandOutput := &bytes.Buffer{}
	command.Stdout = commandOutput
	command.Stderr = commandOutput
	err := command.Start()
	if err != nil {
		return lib.ErrorTestResult(testFile, err)
	}
	timer := time.AfterFunc(timeout, func() {
		command.Process.Kill()
		commandOutput.WriteString(fmt.Sprintf("Killed by testbrain: Timed out after %v", timeout))
	})
	err = command.Wait()
	timer.Stop()

	exitCode, err := getErrorCode(err, command)
	if err != nil {
		return lib.ErrorTestResult(testFile, err)
	}

	return lib.TestResult{
		TestFile: testFile,
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
