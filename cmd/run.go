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
	"sync"
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
	runCmd.PersistentFlags().BoolP("dry-run", "n", false, "Do not actually run the tests")

	viper.BindPFlags(runCmd.PersistentFlags())
}

func runCommandWithViperArgs(cmd *cobra.Command, args []string) error {
	timeoutInSeconds := viper.GetInt("timeout")
	flagJSONOutput := viper.GetBool("json")
	flagVerbose := viper.GetBool("verbose")
	flagTimeout := time.Duration(timeoutInSeconds) * time.Second
	flagInclude := viper.GetString("include")
	flagExclude := viper.GetString("exclude")
	flagDryRun := viper.GetBool("dry-run")
	if len(args) == 0 {
		// No args given, current working directory is assumed
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = []string{cwd}
	}
	return runCommand(args, flagInclude, flagExclude, flagTimeout, flagJSONOutput, flagVerbose, flagDryRun)
}

func runCommand(testFolders []string, includeRe, excludeRe string, timeout time.Duration, jsonOutput, verbose, dryRun bool) error {
	testRoot, testFiles, err := getTestScripts(testFolders, includeRe, excludeRe)
	if err != nil {
		return err
	}
	if !jsonOutput {
		ui.Printf("Found %d test files\n", len(testFiles))
	}
	if dryRun {
		if !jsonOutput {
			ui.Printf("Test root: %s\n", testRoot)
			ui.Printf("Test files:\n")
			for _, testFile := range testFiles {
				ui.Printf("\t%s\n", testFile)
			}
		}
		return nil
	}

	outputIndividualResults := !jsonOutput && verbose
	testResults := runAllTests(testFiles, testRoot, timeout, outputIndividualResults)

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

func getTestScripts(testFolders []string, include, exclude string) (string, []string, error) {
	includeRe, err := regexp.Compile(include)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to include: %s", err)
	}
	excludeRe, err := regexp.Compile(exclude)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to exclude: %s", err)
	}

	var foundTests []string
	for _, testFolderOriginal := range testFolders {
		testFolder, err := filepath.Abs(testFolderOriginal)
		if err != nil {
			return "", nil, fmt.Errorf("Error making %s absolute", testFolderOriginal)
		}
		info, err := os.Stat(testFolder)
		if err != nil {
			return "", nil, fmt.Errorf("Error reading test file %s: %s", testFolder, err)
		}
		if !info.IsDir() {
			// This is an individual test
			if excludeRe.MatchString(testFolder) {
				continue
			}
			foundTests = append(foundTests, testFolder)
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
			foundTests = append(foundTests, path)
			return nil
		})
		if err != nil {
			return "", nil, err
		}
	}

	var commonPrefix string
	if len(testFolders) > 1 {
		commonPrefix, err = lib.CommonPathPrefix(foundTests)
		if err != nil {
			return "", nil, fmt.Errorf("Error getting common prefix for paths: %s", err)
		}
	} else {
		// Only one test folder given; use it as the prefix; unless it's a file,
		// then in which case use its directory
		testFolder, err := filepath.Abs(testFolders[0])
		if err != nil {
			return "", nil, fmt.Errorf("Error finding absolute path of %s: %s", testFolders[0], err)
		}
		info, err := os.Stat(testFolder)
		if err != nil {
			return "", nil, fmt.Errorf("Error stating %s: %s", testFolder, err)
		}
		if info.IsDir() {
			commonPrefix = testFolder
		} else {
			commonPrefix = filepath.Dir(testFolder)
		}
	}
	for i, path := range foundTests {
		relPath, err := filepath.Rel(commonPrefix, path)
		if err != nil {
			return "", nil, fmt.Errorf("Error finding relative path of %s from %s: %s", path, commonPrefix, err)
		}
		foundTests[i] = relPath
	}
	return commonPrefix, foundTests, nil
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
	command := exec.Command(filepath.Join(testFolder, testFile))
	commandOutput := &bytes.Buffer{}
	command.Stdout = commandOutput
	command.Stderr = commandOutput
	err := command.Start()
	if err != nil {
		return lib.ErrorTestResult(testFile, err)
	}
	var timeoutLock sync.Mutex
	timeoutReached := false
	timer := time.AfterFunc(timeout, func() {
		command.Process.Kill()
		timeoutLock.Lock()
		defer timeoutLock.Unlock()
		timeoutReached = true
	})
	err = command.Wait()
	timer.Stop()
	timeoutLock.Lock()
	if timeoutReached {
		commandOutput.WriteString(fmt.Sprintf("Killed by testbrain: Timed out after %v", timeout))
	}
	timeoutLock.Unlock()

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
