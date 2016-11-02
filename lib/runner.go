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

type Runner struct {
	testTargets []string
	includeRe   string
	excludeRe   string
	timeout     time.Duration
	inOrder     bool
	randomSeed  int64
	jsonOutput  bool
	verbose     bool
	dryRun      bool
}

func NewRunner(testFolders []string, flagInclude, flagExclude string, flagTimeout time.Duration,
	flagInOrder bool, flagSeed int64,
	flagJSONOutput, flagVerbose, flagDryRun bool) *Runner {
	r := new(Runner)
	r.testTargets = testFolders
	r.includeRe = flagInclude
	r.excludeRe = flagExclude
	r.timeout = flagTimeout
	r.inOrder = flagInOrder
	r.randomSeed = flagSeed
	r.jsonOutput = flagJSONOutput
	r.verbose = flagVerbose
	r.dryRun = flagDryRun
	return r
}

func (r Runner) RunCommand() error {
	testRoot, testFiles, err := r.getTestScriptsWithOrder()
	if err != nil {
		return err
	}
	if !r.jsonOutput {
		UI.Printf("Found %d test files\n", len(testFiles))
	}
	if r.dryRun {
		if !r.jsonOutput {
			UI.Printf("Test root: %s\n", testRoot)
			UI.Printf("Test files:\n")
			for _, testFile := range testFiles {
				UI.Printf("\t%s\n", testFile)
			}
		}
		return nil
	}

	testResults := r.runAllTests(testFiles, testRoot)

	failedTestResults := r.getFailedTestResults(testResults)
	if r.jsonOutput {
		r.outputResultsJSON(failedTestResults, len(testResults))
	} else {
		r.outputResults(failedTestResults, len(testResults))
	}
	if len(failedTestResults) == 0 {
		return nil
	}
	return fmt.Errorf("%d tests failed", len(failedTestResults))
}

func (r Runner) getTestScriptsWithOrder() (string, []string, error) {
	testRoot, testFiles, err := r.getTestScripts()
	if err != nil {
		return "", nil, err
	}
	sort.Strings(testFiles)
	if !r.inOrder {
		r.shuffleOrder(testFiles)
	}
	return testRoot, testFiles, err
}

func (r Runner) getTestScripts() (string, []string, error) {
	includeRe, err := regexp.Compile(r.includeRe)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to include: %s", err)
	}
	excludeRe, err := regexp.Compile(r.excludeRe)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to exclude: %s", err)
	}

	var foundTests []string
	for _, testFolderOriginal := range r.testTargets {
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
	if len(r.testTargets) > 1 {
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
		testFolder, err := filepath.Abs(r.testTargets[0])
		if err != nil {
			return "", nil, fmt.Errorf("Error finding absolute path of %s: %s", r.testTargets[0], err)
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

func (r Runner) shuffleOrder(list []string) {
	rand.Seed(r.randomSeed)
	// See https://en.wikipedia.org/wiki/Fisher–Yates_shuffle#The_modern_algorithm
	for i := len(list) - 1; i >= 1; i-- {
		source := rand.Intn(i + 1)
		list[i], list[source] = list[source], list[i]
	}
}

func (r Runner) runAllTests(testFiles []string, testFolder string) []TestResult {
	var testResults []TestResult
	for i, testFile := range testFiles {
		if !r.jsonOutput && r.verbose {
			UI.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := r.runSingleTest(testFile, testFolder)
		r.printVerboseSingleTestResult(result)
		testResults = append(testResults, result)
	}
	return testResults
}

func (r Runner) runSingleTest(testFile string, testFolder string) TestResult {
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
	timer := time.AfterFunc(r.timeout, func() {
		command.Process.Kill()
		timeoutLock.Lock()
		defer timeoutLock.Unlock()
		timeoutReached = true
	})
	err = command.Wait()
	timer.Stop()
	timeoutLock.Lock()
	if timeoutReached {
		commandOutput.WriteString(fmt.Sprintf("Killed by testbrain: Timed out after %v", r.timeout))
	}
	timeoutLock.Unlock()

	exitCode, err := r.getErrorCode(err, command)
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

func (r Runner) getErrorCode(err error, command *exec.Cmd) (int, error) {
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

func (r Runner) printVerboseSingleTestResult(result TestResult) {
	if !r.jsonOutput && r.verbose {
		if result.Success {
			Green.Println("OK")
		} else {
			RedBold.Println("FAILED")
			Red.Printf("Output:\n%s\n", result.Output)
		}
	}
}

func (r Runner) getFailedTestResults(testResults []TestResult) []TestResult {
	failedTestResults := []TestResult{}
	for _, result := range testResults {
		if !result.Success {
			failedTestResults = append(failedTestResults, result)
		}
	}
	return failedTestResults
}

func (r Runner) outputResults(failedTestResults []TestResult, nbTestsRan int) {
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
	if !r.inOrder {
		UI.Printf("Seed used: %d\n", r.randomSeed)
	}
}

func (r Runner) outputResultsJSON(failedTestResults []TestResult, nbTestsRan int) {
	displayedSeed := r.randomSeed
	if r.inOrder {
		displayedSeed = -1
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
		Seed:       displayedSeed,
		InOrder:    r.inOrder,
		FailedList: failedTestResults,
	}
	jsonOutput, _ := json.Marshal(jsonOutputStruct)
	UI.Write(jsonOutput)
}
