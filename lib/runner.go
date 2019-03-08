package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"

	"github.com/fatih/color"
)

const (
	unknownExitCode = -1
)

var (
	green      = color.New(color.FgGreen).SprintfFunc()
	greenBold  = color.New(color.FgGreen, color.Bold).SprintfFunc()
	yellowBold = color.New(color.FgYellow, color.Bold).SprintfFunc()
	red        = color.New(color.FgRed).SprintfFunc()
	redBold    = color.New(color.FgRed, color.Bold).SprintfFunc()
)

// RunnerOptions represents options passed to the Runner.
type RunnerOptions struct {
	TestTargets  []string
	IncludeReStr string
	ExcludeReStr string
	Timeout      time.Duration
	InOrder      bool
	RandomSeed   int64
	JSONOutput   bool
	Verbose      bool
	DryRun       bool
}

// Runner runs a series of tests and displays its results.
type Runner struct {
	stderr io.Writer
	stdout io.Writer

	options RunnerOptions

	testResults       []TestResult
	failedTestResults []TestResult
}

// NewRunner constructs a new Runner.
func NewRunner(
	stdout io.Writer,
	stderr io.Writer,
	options RunnerOptions,
) *Runner {
	return &Runner{
		stdout:  stdout,
		stderr:  stderr,
		options: options,
	}
}

// RunCommand is the public entrypoint of the Runner.
// It gathers test scripts, runs them, and displays the result.
func (r *Runner) RunCommand() error {
	testRoot, testFiles, err := r.getTestScriptsWithOrder()
	if err != nil {
		return err
	}
	if !r.options.JSONOutput {
		fmt.Fprintf(r.stdout, "Found %d test files\n", len(testFiles))
		if !r.options.InOrder {
			fmt.Fprintf(r.stdout, "Using seed: %d\n", r.options.RandomSeed)
		}
	}
	if r.options.DryRun {
		if !r.options.JSONOutput {
			fmt.Fprintf(r.stdout, "Test root: %s\n", testRoot)
			fmt.Fprintf(r.stdout, "Test files:\n")
			for _, testFile := range testFiles {
				fmt.Fprintf(r.stdout, "\t%s\n", testFile)
			}
		}
		return nil
	}

	r.runAllTests(testFiles, testRoot)
	r.collectFailedTestResults()
	if r.options.JSONOutput {
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
	if !r.options.InOrder {
		r.shuffleOrder(testFiles)
	}
	return testRoot, testFiles, err
}

func (r *Runner) getTestScripts() (string, []string, error) {
	includeRe, err := regexp.Compile(r.options.IncludeReStr)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to include: %s", err)
	}
	excludeRe, err := regexp.Compile(r.options.ExcludeReStr)
	if err != nil {
		return "", nil, fmt.Errorf("Error parsing files to exclude: %s", err)
	}

	var foundTests []string
	for _, testFolderOriginal := range r.options.TestTargets {
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
	if len(r.options.TestTargets) > 1 {
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
		testFolder, err := filepath.Abs(r.options.TestTargets[0])
		if err != nil {
			return "", nil, fmt.Errorf("Error finding absolute path of %s: %s", r.options.TestTargets[0], err)
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
	rand.Seed(r.options.RandomSeed)
	// See https://en.wikipedia.org/wiki/Fisher–Yates_shuffle#The_modern_algorithm.
	for i := len(list) - 1; i >= 1; i-- {
		source := rand.Intn(i + 1)
		list[i], list[source] = list[source], list[i]
	}
}

func (r *Runner) runAllTests(testFiles []string, testFolder string) {
	for i, testFile := range testFiles {
		if !r.options.JSONOutput {
			fmt.Fprintf(r.stdout, "Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}
		result := r.runSingleTest(testFile, testFolder)
		r.printVerboseSingleTestResult(result)
		r.testResults = append(r.testResults, result)
	}
}

func (r *Runner) runSingleTest(testFile string, testFolder string) TestResult {
	testPath := filepath.Join(testFolder, testFile)

	command := exec.Command(testPath)

	testResult := TestResult{
		TestFile: testFile,
	}

	if r.options.Verbose {
		command.Stdout = r.stdout
		command.Stderr = r.stderr
	} else {
		var buf bytes.Buffer
		command.Stdout = &buf
		command.Stderr = &buf
		testResult.Output = &buf
	}

	// Propagate timeout information from brain to script, via the environment of the script.
	env := os.Environ()
	env = append(env, fmt.Sprintf("TESTBRAIN_TIMEOUT=%v", r.options.Timeout.Seconds()))
	command.Env = env

	err := command.Start()
	if err != nil {
		fmt.Fprintf(r.stderr, "Test failed: %v", err)
		testResult.Success = false
		testResult.ExitCode = unknownExitCode
		return testResult
	}

	done := make(chan error)
	go func() {
		done <- command.Wait()
		close(done)
	}()

	timeout := time.After(r.options.Timeout)

	select {
	case <-timeout:
		command.Process.Kill()
		fmt.Fprintf(r.stderr, "Killed by testbrain: Timed out after %v\n", r.options.Timeout)
		testResult.Success = false
		testResult.ExitCode = unknownExitCode
	case err = <-done:
		testResult.ExitCode, err = r.getErrorCode(err, command)
		if err != nil {
			fmt.Fprintf(r.stderr, "Test failed: %v", err)
			testResult.Success = false
			testResult.ExitCode = unknownExitCode
			return testResult
		}
		testResult.Success = command.ProcessState.Success()
	}

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
			status, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				return status.ExitStatus(), nil
			}
		}

		// There is an error but it's not an ExitError.
		// Something other than the test script failed, bubble up the error.
		return unknownExitCode, err
	}

	// The test script failed, but without an error.
	return unknownExitCode, nil
}

func (r *Runner) printVerboseSingleTestResult(result TestResult) {
	if !r.options.JSONOutput {
		if result.Success {
			fmt.Fprintf(r.stdout, "%s: %s\n", greenBold("PASSED"), result.TestFile)
		} else {
			fmt.Fprintf(r.stderr, "%s: %s\n", redBold("FAILED"), result.TestFile)
			if !r.options.Verbose {
				fmt.Fprintln(r.stderr, "Test output:")
				io.Copy(r.stderr, result.Output)
			}
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
		fmt.Fprintf(r.stderr, redBold("%s: Failed with exit code %d\n", failedResult.TestFile, failedResult.ExitCode))
	}
	nbTestsFailed := len(r.failedTestResults)
	nbTestsPassed := len(r.testResults) - nbTestsFailed
	summaryString := fmt.Sprintf("\nTests complete: %d Passed, %d Failed", nbTestsPassed, nbTestsFailed)
	if nbTestsFailed > 0 {
		fmt.Fprintln(r.stderr, redBold(summaryString))
	} else {
		fmt.Fprintln(r.stdout, greenBold(summaryString))
	}
	if !r.options.InOrder {
		fmt.Fprintf(r.stdout, "Seed used: %d\n", r.options.RandomSeed)
	}
}

func (r *Runner) outputResultsJSON() {
	displayedSeed := r.options.RandomSeed
	if r.options.InOrder {
		displayedSeed = unknownExitCode
	}
	nbTestsFailed := len(r.failedTestResults)
	nbTestsPassed := len(r.testResults) - nbTestsFailed

	// This is the only place where we need this struct, so anonymous struct seems appropriate.
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
		InOrder:    r.options.InOrder,
		FailedList: r.failedTestResults,
	}

	jsonEncoder := json.NewEncoder(r.stdout)
	err := jsonEncoder.Encode(jsonOutputStruct)
	if err != nil {
		fmt.Fprintln(r.stderr, redBold("Error trying to marshal JSON output"))
	}
}

// TestResult contains the result of a single test script.
type TestResult struct {
	TestFile string    `json:"filename"`
	Success  bool      `json:"success"`
	ExitCode int       `json:"exitcode"`
	Output   io.Reader `json:"-"`
}
