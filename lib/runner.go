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
	unknownExitCode  = -1
	skipTestExitCode = 99
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

	passedResults, skippedResults, failedResults := r.runAllTests(testFiles, testRoot)
	if r.options.JSONOutput {
		r.outputResultsJSON(passedResults, skippedResults, failedResults)
	} else {
		r.outputResults(passedResults, skippedResults, failedResults)
	}
	if len(failedResults) == 0 {
		return nil
	}
	return fmt.Errorf("%d tests failed", len(failedResults))
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

func (r *Runner) runAllTests(testFiles []string, testFolder string) ([]PassedResult, []SkippedResult, []FailedResult) {
	passedResults := make([]PassedResult, 0)
	skippedResults := make([]SkippedResult, 0)
	failedResults := make([]FailedResult, 0)
	for i, testFile := range testFiles {
		if !r.options.JSONOutput {
			fmt.Fprintf(r.stdout, "Running test %s (%d/%d)\n", testFile, i+1, len(testFiles))
		}

		var cmdStdout, cmdStderr io.Writer
		var outputBuf bytes.Buffer
		if r.options.Verbose {
			cmdStdout = r.stdout
			cmdStderr = r.stderr
		} else {
			cmdStdout = &outputBuf
			cmdStderr = &outputBuf
		}
		exitCode := r.runSingleTest(testFile, testFolder, cmdStdout, cmdStderr)

		if exitCode == skipTestExitCode {
			result := SkippedResult{testFile}
			skippedResults = append(skippedResults, result)
			fmt.Fprintln(r.stdout, result)
		} else if exitCode == 0 {
			result := PassedResult{testFile}
			passedResults = append(passedResults, result)
			fmt.Fprintln(r.stdout, result)
		} else {
			result := FailedResult{
				TestResult{testFile},
				exitCode,
			}
			failedResults = append(failedResults, result)
			fmt.Fprintln(r.stdout, result)
			if !r.options.Verbose {
				fmt.Fprintln(r.stdout, "Test output:")
				io.Copy(r.stdout, &outputBuf)
			}
		}
	}
	return passedResults, skippedResults, failedResults
}

func (r *Runner) runSingleTest(testFile string, testFolder string, cmdStdout, cmdStderr io.Writer) (exitCode int) {
	testPath := filepath.Join(testFolder, testFile)

	command := exec.Command(testPath)
	command.Stdout = cmdStdout
	command.Stderr = cmdStderr

	// Propagate timeout information from brain to script, via the environment of the script.
	env := os.Environ()
	env = append(env, fmt.Sprintf("TESTBRAIN_TIMEOUT=%v", r.options.Timeout.Seconds()))
	command.Env = env

	err := command.Start()
	if err != nil {
		fmt.Fprintf(r.stderr, "Test failed: %v", err)
		return unknownExitCode
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
		return unknownExitCode
	case err = <-done:
		exitCode, err = r.getErrorCode(err, command)
		if err != nil {
			fmt.Fprintf(r.stderr, "Test failed: %v", err)
			return
		}
	}
	return
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

func (r *Runner) outputResults(passedResults []PassedResult, skippedResults []SkippedResult, failedResults []FailedResult) {
	summaryString := fmt.Sprintf(
		"Tests complete: %d Passed, %d Skipped, %d Failed",
		len(passedResults), len(skippedResults), len(failedResults))
	if len(failedResults) > 0 {
		fmt.Fprintf(r.stdout, "%s\n\n", redBold(summaryString))
	} else {
		fmt.Fprintf(r.stdout, "%s\n\n", greenBold(summaryString))
	}

	if len(skippedResults) > 0 {
		fmt.Fprintln(r.stdout, "  Skipped tests:")
		for _, result := range skippedResults {
			fmt.Fprintf(r.stdout, "    %s\n", result.TestFile)
		}
		fmt.Fprintf(r.stdout, "\n")
	}

	if len(failedResults) > 0 {
		fmt.Fprintln(r.stdout, "  Failed tests:")
		for _, result := range failedResults {
			fmt.Fprintf(r.stdout, "    %s with exit code %d\n", result.TestFile, result.ExitCode)
		}
		fmt.Fprintf(r.stdout, "\n")
	}
}

func (r *Runner) outputResultsJSON(passedResults []PassedResult, skippedResults []SkippedResult, failedResults []FailedResult) {
	displayedSeed := r.options.RandomSeed
	if r.options.InOrder {
		displayedSeed = unknownExitCode
	}

	// This is the only place where we need this struct, so anonymous struct seems appropriate.
	jsonOutputStruct := struct {
		Passed      int             `json:"passed"`
		Skipped     int             `json:"skipped"`
		Failed      int             `json:"failed"`
		Seed        int64           `json:"seed"`
		InOrder     bool            `json:"inOrder"`
		PassedList  []PassedResult  `json:"passedList"`
		SkippedList []SkippedResult `json:"skippedList"`
		FailedList  []FailedResult  `json:"failedList"`
	}{
		Passed:      len(passedResults),
		Skipped:     len(skippedResults),
		Failed:      len(failedResults),
		Seed:        displayedSeed,
		InOrder:     r.options.InOrder,
		PassedList:  passedResults,
		SkippedList: skippedResults,
		FailedList:  failedResults,
	}

	jsonEncoder := json.NewEncoder(r.stdout)
	err := jsonEncoder.Encode(jsonOutputStruct)
	if err != nil {
		fmt.Fprintln(r.stderr, redBold("Error trying to marshal JSON output"))
	}
}

// TestResult contains the result of a single test script.
type TestResult struct {
	TestFile string `json:"filename"`
}

// PassedResult is a type for a test result that passed.
type PassedResult TestResult

func (result PassedResult) String() string {
	return fmt.Sprintf("%s: %s\n", greenBold("PASSED"), result.TestFile)
}

// SkippedResult is a type for a test result that skipped.
type SkippedResult TestResult

func (result SkippedResult) String() string {
	return fmt.Sprintf("%s: %s\n", yellowBold("SKIPPED"), result.TestFile)
}

// FailedResult is a type for a test result that failed.
type FailedResult struct {
	TestResult
	ExitCode int `json:"exitcode"`
}

func (result FailedResult) String() string {
	return fmt.Sprintf("%s: %s\n", redBold("FAILED"), result.TestFile)
}
