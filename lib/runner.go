package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"
)

// Runner runs a series of tests and displays its results.
type Runner struct {
	// Flags
	testTargets  []string
	includeReStr string
	excludeReStr string
	timeout      time.Duration
	inOrder      bool
	randomSeed   int64
	jsonOutput   bool
	verbose      bool
	dryRun       bool

	testResults       []TestResult
	failedTestResults []TestResult
}

// NewRunner constructs a new Runner.
func NewRunner(
	testTargets []string,
	includeReStr string,
	excludeReStr string,
	timeout time.Duration,
	inOrder bool,
	randomSeed int64,
	jsonOutput bool,
	verbose bool,
	dryRun bool,
) *Runner {
	return &Runner{
		testTargets:  testTargets,
		includeReStr: includeReStr,
		excludeReStr: excludeReStr,
		timeout:      timeout,
		inOrder:      inOrder,
		randomSeed:   randomSeed,
		jsonOutput:   jsonOutput,
		verbose:      verbose,
		dryRun:       dryRun,
	}
}

// RunCommand is the public entrypoint of the Runner.
// It gathers test scripts, runs them, and displays the result.
func (r *Runner) RunCommand() error {
	testRoot, testFiles, err := r.getTestScriptsWithOrder()
	if err != nil {
		return err
	}
	if !r.jsonOutput {
		UI.Printf("Found %d test files\n", len(testFiles))
		if !r.inOrder {
			UI.Printf("Using seed: %d\n", r.randomSeed)
		}
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

	r.runAllTests(testFiles, testRoot)
	r.collectFailedTestResults()
	if r.jsonOutput {
		r.outputResultsJSON()
	} else {
		r.outputResults()
	}
	if len(r.failedTestResults) == 0 {
		return nil
	}
	return fmt.Errorf("%d tests failed", len(r.failedTestResults))
}

func (r *Runner) getTestScriptsWithOrder() (string, []string, error) {
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

func (r *Runner) getTestScripts() (string, []string, error) {
	includeRe, err := regexp.Compile(r.includeReStr)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to include: %s", err)
	}
	excludeRe, err := regexp.Compile(r.excludeReStr)
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
		if len(foundTests) != 1 {
			// None or more than one test found.
			commonPrefix, err = CommonPathPrefix(foundTests)
			if err != nil {
				return "", nil, fmt.Errorf("Error getting common prefix for paths: %s", err)
			}
		} else {
			// Only one test found of many specified; use its parent as the prefix.
			// Otherwise CommonPathPrefix() would give a silly result.
			commonPrefix = filepath.Dir(foundTests[0])
		}
	} else {
		// Only one test folder given; use it as the prefix; unless it's a file,
		// then in which case use its directory.
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

func (r *Runner) shuffleOrder(list []string) {
	rand.Seed(r.randomSeed)
	// See https://en.wikipedia.org/wiki/Fisher–Yates_shuffle#The_modern_algorithm.
	for i := len(list) - 1; i >= 1; i-- {
		source := rand.Intn(i + 1)
		list[i], list[source] = list[source], list[i]
	}
}

func (r *Runner) runAllTests(testFiles []string, testFolder string) {
	for i, testFile := range testFiles {
		if !r.jsonOutput && r.verbose {
			UI.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := r.runSingleTest(testFile, testFolder)
		r.printVerboseSingleTestResult(result)
		r.testResults = append(r.testResults, result)
	}
}

func (r *Runner) runSingleTest(testFile string, testFolder string) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	testPath := filepath.Join(testFolder, testFile)

	command := exec.CommandContext(ctx, testPath)

	commandOutput := &bytes.Buffer{}
	command.Stdout = commandOutput
	command.Stderr = commandOutput

	// Propagate timeout information from brain to script, via the environment of the script.
	env := os.Environ()
	env = append(env, fmt.Sprintf("TESTBRAIN_TIMEOUT=%d", int(r.timeout.Seconds())))
	command.Env = env

	err := command.Start()
	if err != nil {
		return ErrorTestResult(testFile, err)
	}

	done := make(chan error)
	go func() {
		done <- command.Wait()
	}()

	timeout := time.After(r.timeout)

	testResult := TestResult{
		TestFile: testFile,
	}

	select {
	case <-timeout:
		command.Process.Kill()
		commandOutput.WriteString(fmt.Sprintf("Killed by testbrain: Timed out after %v", r.timeout))
		testResult.Success = false
		testResult.ExitCode = -1
	case err = <-done:
		testResult.ExitCode, err = r.getErrorCode(err, command)
		if err != nil {
			return ErrorTestResult(testFile, err)
		}
		testResult.Success = command.ProcessState.Success()
	}
	testResult.Output = string(commandOutput.Bytes())

	return testResult
}

func (r *Runner) getErrorCode(err error, command *exec.Cmd) (int, error) {
	if command.ProcessState.Success() {
		// Not exactly necessary, since we can check Success(),
		// but more correct than saying status code is unknown.
		return 0, nil
	}
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			// The program has exited with an ExitError, we can get the error code from WaitStatus.
			// See https://golang.org/pkg/os/#ProcessState.Sys.
			// Although the docs mention syscall.WaitStatus works on Unix, it seems to work on Windows
			// too.
			status, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				return status.ExitStatus(), nil
			}
		}

		// There is an error but it's not an ExitError.
		// Something other than the test script failed, bubble up the error.
		return UnknownExitCode, err
	}

	// The test script failed, but without an error.
	return UnknownExitCode, nil
}

func (r *Runner) printVerboseSingleTestResult(result TestResult) {
	if !r.jsonOutput && r.verbose {
		if result.Success {
			Green.Println("OK")
		} else {
			RedBold.Println("FAILED")
			Red.Printf("Output:\n%s\n", result.Output)
		}
	}
}

func (r *Runner) collectFailedTestResults() {
	for _, result := range r.testResults {
		if !result.Success {
			r.failedTestResults = append(r.failedTestResults, result)
		}
	}
}

func (r *Runner) outputResults() {
	for _, failedResult := range r.failedTestResults {
		RedBold.Printf("%s: Failed with exit code %d\n", failedResult.TestFile, failedResult.ExitCode)
	}
	nbTestsFailed := len(r.failedTestResults)
	nbTestsPassed := len(r.testResults) - nbTestsFailed
	summaryString := fmt.Sprintf("\nTests complete: %d Passed, %d Failed", nbTestsPassed, nbTestsFailed)
	if nbTestsFailed > 0 {
		RedBold.Println(summaryString)
	} else {
		GreenBold.Println(summaryString)
	}
	if !r.inOrder {
		UI.Printf("Seed used: %d\n", r.randomSeed)
	}
}

func (r *Runner) outputResultsJSON() {
	displayedSeed := r.randomSeed
	if r.inOrder {
		displayedSeed = -1
	}
	nbTestsFailed := len(r.failedTestResults)
	nbTestsPassed := len(r.testResults) - nbTestsFailed

	// This is the only place where we need this struct, so anonymous struct seems appropriate
	jsonOutputStruct := struct {
		Passed     int          `json:"passed"`
		Failed     int          `json:"failed"`
		Seed       int64        `json:"seed"`
		InOrder    bool         `json:"inOrder"`
		FailedList []TestResult `json:"failedList"`
	}{
		Passed:     nbTestsPassed,
		Failed:     nbTestsFailed,
		Seed:       displayedSeed,
		InOrder:    r.inOrder,
		FailedList: r.failedTestResults,
	}
	jsonOutput, err := json.Marshal(jsonOutputStruct)
	if err != nil {
		RedBold.Println("Error trying to marshal JSON output")
	}
	UI.Write(jsonOutput)
}
