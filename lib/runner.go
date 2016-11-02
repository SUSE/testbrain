package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"syscall"
	"time"
)

func RunCommand(testFolders []string, includeRe, excludeRe string,
	timeout time.Duration,
	jsonOutput, verbose,
	inOrder bool, randomSeed int64,
	dryRun bool) error {
	testRoot, testFiles, err := getTestScriptsWithOrder(testFolders, includeRe, excludeRe, inOrder, randomSeed)
	if err != nil {
		return err
	}
	if !jsonOutput {
		UI.Printf("Found %d test files\n", len(testFiles))
	}
	if dryRun {
		if !jsonOutput {
			UI.Printf("Test root: %s\n", testRoot)
			UI.Printf("Test files:\n")
			for _, testFile := range testFiles {
				UI.Printf("\t%s\n", testFile)
			}
		}
		return nil
	}

	outputIndividualResults := !jsonOutput && verbose
	testResults := runAllTests(testFiles, testRoot, timeout, outputIndividualResults)

	failedTestResults := getFailedTestResults(testResults)
	if jsonOutput {
		outputResultsJSON(failedTestResults, len(testResults), inOrder, randomSeed)
	} else {
		outputResults(failedTestResults, len(testResults), inOrder, randomSeed)
	}
	if len(failedTestResults) == 0 {
		return nil
	}
	return fmt.Errorf("%d tests failed", len(failedTestResults))
}

func getTestScriptsWithOrder(testFolders []string, include, exclude string, inOrder bool, randomSeed int64) (string, []string, error) {
	testRoot, testFiles, err := getTestScripts(testFolders, include, exclude)
	if err != nil {
		return "", nil, err
	}
	sort.Strings(testFiles)
	if !inOrder {
		shuffleOrder(testFiles, randomSeed)
	}
	return testRoot, testFiles, err
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
			if !includeRe.MatchString(testFolder) {
				continue
			}
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
		if len(foundTests) > 1 {
			commonPrefix, err = CommonPathPrefix(foundTests)
			if err != nil {
				return "", nil, fmt.Errorf("Error getting common prefix for paths: %s", err)
			}
		} else {
			// Only one test found of many specified; use its parent as the prefix
			// Otherwise CommonPathPrefix() would give a silly result
			commonPrefix = filepath.Dir(foundTests[0])
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

func shuffleOrder(list []string, randomSeed int64) {
	rand.Seed(randomSeed)
	// See https://en.wikipedia.org/wiki/Fisher–Yates_shuffle#The_modern_algorithm
	for i := len(list) - 1; i >= 1; i-- {
		source := rand.Intn(i + 1)
		list[i], list[source] = list[source], list[i]
	}
}

func runAllTests(testFiles []string, testFolder string,
	timeout time.Duration, outputIndividualResults bool) []TestResult {
	var testResults []TestResult
	for i, testFile := range testFiles {
		if outputIndividualResults {
			UI.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := runSingleTest(testFile, testFolder, timeout)
		printVerboseSingleTestResult(result, outputIndividualResults)
		testResults = append(testResults, result)
	}
	return testResults
}

func runSingleTest(testFile string, testFolder string, timeout time.Duration) TestResult {
	command := exec.Command(filepath.Join(testFolder, testFile))
	commandOutput := &bytes.Buffer{}
	command.Stdout = commandOutput
	command.Stderr = commandOutput
	err := command.Start()
	if err != nil {
		return ErrorTestResult(testFile, err)
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
		return ErrorTestResult(testFile, err)
	}

	return TestResult{
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
		return UnknownExitCode, err
	}

	// The test script failed, but without an error
	return UnknownExitCode, nil
}

func printVerboseSingleTestResult(result TestResult, outputIndividualResults bool) {
	if outputIndividualResults {
		if result.Success {
			Green.Println("OK")
		} else {
			RedBold.Println("FAILED")
			Red.Printf("Output:\n%s\n", result.Output)
		}
	}
}

func getFailedTestResults(testResults []TestResult) []TestResult {
	failedTestResults := []TestResult{}
	for _, result := range testResults {
		if !result.Success {
			failedTestResults = append(failedTestResults, result)
		}
	}
	return failedTestResults
}

func outputResults(failedTestResults []TestResult, nbTestsRan int, inOrder bool, randomSeed int64) {
	for _, failedResult := range failedTestResults {
		RedBold.Printf("%s: Failed with exit code %d\n", failedResult.TestFile, failedResult.ExitCode)
	}
	nbTestsPassed := nbTestsRan - len(failedTestResults)
	summaryString := fmt.Sprintf("\nTests complete: %d Passed, %d Failed", nbTestsPassed, len(failedTestResults))
	if len(failedTestResults) > 0 {
		RedBold.Println(summaryString)
	} else {
		GreenBold.Println(summaryString)
	}
	if !inOrder {
		UI.Printf("Seed used: %d\n", randomSeed)
	}
}

func outputResultsJSON(failedTestResults []TestResult, nbTestsRan int, inOrder bool, randomSeed int64) {
	if inOrder {
		randomSeed = -1
	}
	// This is the only place where we need this struct, so anonymous struct seems appropriate
	jsonOutputStruct := struct {
		Passed     int          `json:"passed"`
		Failed     int          `json:"failed"`
		Seed       int64        `json:"seed"`
		InOrder    bool         `json:"inOrder"`
		FailedList []TestResult `json:"failedList"`
	}{
		Passed:     nbTestsRan - len(failedTestResults),
		Failed:     len(failedTestResults),
		Seed:       randomSeed,
		InOrder:    inOrder,
		FailedList: failedTestResults,
	}
	jsonOutput, _ := json.Marshal(jsonOutputStruct)
	UI.Write(jsonOutput)
}
