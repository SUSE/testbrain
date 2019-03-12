package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/fatih/color"
)

var (
	defaultTimeout = 300 * time.Second
	defaultInclude = "_test\\.sh$"
	defaultExclude = "^$"
	defaultSeed    = time.Now().UnixNano()

	redBoldString = color.New(color.FgRed, color.Bold).SprintfFunc()
)

func setupDefaultRunner(stdout io.Writer, stderr io.Writer) *Runner {
	options := RunnerOptions{
		TestTargets:  []string{},
		IncludeReStr: defaultInclude,
		ExcludeReStr: defaultExclude,
		Timeout:      defaultTimeout,
		InOrder:      false,
		RandomSeed:   defaultSeed,
		JSONOutput:   false,
		Verbose:      false,
		DryRun:       false,
	}
	runner := NewRunner(
		stdout,
		stderr,
		options,
	)
	return runner
}

func setupPassedTestResults() []PassedResult {
	return []PassedResult{
		PassedResult{
			TestFile: "testfile-success-1",
		},
		PassedResult{
			TestFile: "testfile-success-2",
		},
	}
}

func setupSkippedTestResults() []SkippedResult {
	return []SkippedResult{
		SkippedResult{
			TestFile: "testfile-skip-1",
		},
		SkippedResult{
			TestFile: "testfile-skip-2",
		},
	}
}

func setupFailedTestResults() []FailedResult {
	return []FailedResult{
		FailedResult{
			TestResult: TestResult{
				TestFile: "testfile-failure-1",
			},
			ExitCode: 1,
		},
		FailedResult{
			TestResult: TestResult{
				TestFile: "testfile-failure-2",
			},
			ExitCode: 2,
		},
	}
}

func TestGetTestScripts(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.options.TestTargets = []string{testFolder}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root '%s' was not '%s'", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh", "001_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFilters(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.options.TestTargets = []string{testFolder}
	r.options.IncludeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root '%s' was not '%s'", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersMultiFolder(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.TestTargets = []string{testFolderA, testFolderB}
	r.options.IncludeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	expectedRoot := "testdata"
	if expectedRoot != filepath.Base(testRoot) {
		t.Errorf("Test root %s was not %s", testRoot, expectedRoot)
	}
	expected := []string{
		"testfolder1/000_script_test.sh",
		"testfolder3-many-tests/000_script_test.sh",
	}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersNoMatch(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.options.TestTargets = []string{testFolder}
	r.options.IncludeReStr = "002" // testfolder1 has only 000 and 001

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}

	if len(testScripts) > 0 {
		t.Errorf("\nExpected:\n[]\nHave:\n%v\n", testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersMultiFolderNoMatch(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.TestTargets = []string{testFolderA, testFolderB}
	r.options.IncludeReStr = "005"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	expectedRoot := ""
	if expectedRoot != testRoot {
		t.Errorf("Test root '%s' was not '%s'", testRoot, expectedRoot)
	}
	if len(testScripts) > 0 {
		t.Errorf("\nExpected:\n[]\nHave:\n%v\n", testScripts)
	}
}

func TestGetTestScripts_ExcludeFilters(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	r.options.TestTargets = []string{testFolder}
	r.options.ExcludeReStr = "001"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_NestedDirectories(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder2")
	r.options.TestTargets = []string{testFolder}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"nested_directory/test_file_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_OnlyOneFile(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/success")
	testPath := filepath.Join(testFolder, "hello_world_test.sh")
	r.options.TestTargets = []string{testPath}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test script: %s", err)
	}
	if testRoot != testFolder {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"hello_world_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_NotIncludedExplicit(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	r.options.TestTargets = testFiles
	r.options.IncludeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScripts_OnlyOneResult(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	r.options.TestTargets = testFiles
	r.options.ExcludeReStr = "001"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Errorf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScriptsWithOrder_SameSeedSameOrder(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.TestTargets = []string{testFolder}

	_, testScripts1, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	_, testScripts2, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	if !reflect.DeepEqual(testScripts1, testScripts2) {
		t.Errorf("Different results using the seed %d: %v and %v\n", defaultSeed, testScripts1, testScripts2)
	}
}

func TestGetTestScriptsWithOrder_SpecificSeed(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.TestTargets = []string{testFolder}
	r.options.RandomSeed = 42

	_, testScripts, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	expected := []string{"001_script_test.sh", "004_script_test.sh", "002_script_test.sh", "003_script_test.sh", "000_script_test.sh"}
	if !reflect.DeepEqual(expected, testScripts) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestGetTestScriptsWithOrder_InOrder(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.TestTargets = []string{testFolder}
	r.options.InOrder = true

	_, testScripts, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Errorf("Error getting test scripts: %s", err)
	}
	expected := []string{"000_script_test.sh", "001_script_test.sh", "002_script_test.sh", "003_script_test.sh", "004_script_test.sh"}
	if !reflect.DeepEqual(expected, testScripts) {
		t.Errorf("\nExpected:\n%v\nHave:\n%v\n", expected, testScripts)
	}
}

func TestShuffleOrder(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFiles := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	originalOrder := make([]string, len(testFiles))
	copy(originalOrder, testFiles)
	r.shuffleOrder(testFiles)
	if reflect.DeepEqual(testFiles, originalOrder) {
		t.Errorf("Shuffle order did not change: %v\n", testFiles)
	}
}

func TestRunSingleTestSuccess(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/success")
	testFile := "hello_world_test.sh"
	exitCode := r.runSingleTest(testFile, testFolder, ioutil.Discard, ioutil.Discard)
	if exitCode != 0 {
		t.Errorf("\nExpected ExitCode: %v\nHave: %v\n", 0, exitCode)
	}
}

func TestRunSingleTestFailure(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	testFolder, _ := filepath.Abs("../testdata/failure")
	testFile := "failure_test.sh"
	exitCode := r.runSingleTest(testFile, testFolder, ioutil.Discard, ioutil.Discard)
	if exitCode != 42 {
		t.Errorf("\nExpected ExitCode: %v\nHave: %v\n", 42, exitCode)
	}
}

func TestRunSingleTestTimeout(t *testing.T) {
	t.Parallel()

	testFile := "timeout_test.sh"

	tests := []struct {
		title            string
		verbose          bool
		expectedExitCode int
		expectedStdout   string
		expectedStderr   string
	}{
		{
			title:            "verbose",
			verbose:          true,
			expectedExitCode: unknownExitCode,
			expectedStdout: "Timeout = 1\n" +
				"Long running process...\n",
			expectedStderr: "Killed by testbrain: Timed out after 1s\n",
		},
		{
			title:            "non-verbose",
			verbose:          false,
			expectedExitCode: unknownExitCode,
			expectedStdout: "Timeout = 1\n" +
				"Long running process...\n",
			expectedStderr: "Killed by testbrain: Timed out after 1s\n",
		},
	}

	testFolder, _ := filepath.Abs("../testdata")

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var stdout concurrentBuffer
			var stderr concurrentBuffer
			r := setupDefaultRunner(&stdout, &stderr)
			r.options.Timeout = 1 * time.Second
			r.options.Verbose = tt.verbose

			exitCode := r.runSingleTest(testFile, testFolder, &stdout, &stderr)
			if exitCode != tt.expectedExitCode {
				t.Errorf("\nExpected ExitCode: %v\nHave: %v\n", tt.expectedExitCode, exitCode)
			}

			stdoutBytes, err := ioutil.ReadAll(&stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(&stderr)
			if err != nil {
				t.Fatal(err)
			}
			if stderrStr := string(stderrBytes); stderrStr != tt.expectedStderr {
				t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", tt.expectedStderr, stderrStr)
			}
		})
	}
}

func TestOutputResults(t *testing.T) {
	var stdout concurrentBuffer
	var stderr concurrentBuffer
	r := setupDefaultRunner(&stdout, &stderr)
	r.options.RandomSeed = 42
	passedResults := setupPassedTestResults()
	skippedResults := setupSkippedTestResults()
	failedResults := setupFailedTestResults()

	r.outputResults(passedResults, skippedResults, failedResults)
	expectedStdout := "Tests complete: 2 Passed, 2 Skipped, 2 Failed\n\n" +
		"  Skipped tests:\n" +
		"    testfile-skip-1\n" +
		"    testfile-skip-2\n\n" +
		"  Failed tests:\n" +
		"    testfile-failure-1 with exit code 1\n" +
		"    testfile-failure-2 with exit code 2\n\n"
	expectedStderr := ""
	stdoutBytes, err := ioutil.ReadAll(&stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(&stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", expectedStderr, stderrStr)
	}
}

func TestOutputResultsJSON(t *testing.T) {
	var stdout concurrentBuffer
	var stderr concurrentBuffer
	r := setupDefaultRunner(&stdout, &stderr)
	r.options.RandomSeed = 42
	passedResults := setupPassedTestResults()
	skippedResults := setupSkippedTestResults()
	failedResults := setupFailedTestResults()

	r.outputResultsJSON(passedResults, skippedResults, failedResults)
	expectedStdout := "{" +
		`"passed":2,` +
		`"skipped":2,` +
		`"failed":2,` +
		`"seed":42,` +
		`"inOrder":false,` +
		`"passedList":[{"filename":"testfile-success-1"},{"filename":"testfile-success-2"}],` +
		`"skippedList":[{"filename":"testfile-skip-1"},{"filename":"testfile-skip-2"}],` +
		`"failedList":[{"filename":"testfile-failure-1","exitcode":1},{"filename":"testfile-failure-2","exitcode":2}]` +
		"}\n"
	expectedStderr := ""
	stdoutBytes, err := ioutil.ReadAll(&stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(&stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", expectedStderr, stderrStr)
	}
}

func TestRunCommand(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/mixed")
	seed := int64(1552072438299530183)

	tests := []struct {
		title          string
		verbose        bool
		expectedStdout string
		expectedStderr string
	}{
		{
			title:   "Verbose",
			verbose: true,
			expectedStdout: "Found 3 test files\n" +
				fmt.Sprintf("Using seed: %d\n", seed) +
				"Running test fail_test.sh (1/3)\n" +
				"Goodbye World!\n" +
				"FAILED: fail_test.sh\n\n" +
				"Running test success_test.sh (2/3)\n" +
				"Hello World!\n" +
				"PASSED: success_test.sh\n\n" +
				"Running test skip_test.sh (3/3)\n" +
				"Something stdout\n" +
				"SKIPPED: skip_test.sh\n\n" +
				"Tests complete: 1 Passed, 1 Skipped, 1 Failed\n\n" +
				"  Skipped tests:\n" +
				"    skip_test.sh\n\n" +
				"  Failed tests:\n" +
				"    fail_test.sh with exit code 42\n\n",
			expectedStderr: "Something stderr\n",
		},
		{
			title:   "Non-verbose",
			verbose: false,
			expectedStdout: "Found 3 test files\n" +
				fmt.Sprintf("Using seed: %d\n", seed) +
				"Running test fail_test.sh (1/3)\n" +
				"FAILED: fail_test.sh\n\n" +
				"Test output:\n" +
				"Goodbye World!\n" +
				"Running test success_test.sh (2/3)\n" +
				"PASSED: success_test.sh\n\n" +
				"Running test skip_test.sh (3/3)\n" +
				"SKIPPED: skip_test.sh\n\n" +
				"Tests complete: 1 Passed, 1 Skipped, 1 Failed\n\n" +
				"  Skipped tests:\n" +
				"    skip_test.sh\n\n" +
				"  Failed tests:\n" +
				"    fail_test.sh with exit code 42\n\n",
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var stdout concurrentBuffer
			var stderr concurrentBuffer
			r := setupDefaultRunner(&stdout, &stderr)
			r.options.RandomSeed = seed
			r.options.Verbose = tt.verbose
			r.options.TestTargets = []string{testFolder}
			err := r.RunCommand()
			if err == nil {
				t.Errorf("Expected an error, got nothing")
			}
			stdoutBytes, err := ioutil.ReadAll(&stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(&stderr)
			if err != nil {
				t.Fatal(err)
			}
			if stderrStr := string(stderrBytes); stderrStr != tt.expectedStderr {
				t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", tt.expectedStderr, stderrStr)
			}
		})
	}
}

func TestRunCommandFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	r := setupDefaultRunner(ioutil.Discard, ioutil.Discard)
	r.options.TestTargets = []string{testFolder}

	err := r.RunCommand()
	if err == nil {
		t.Fatal("Expected to get an error, got 'nil'")
	}
}

type concurrentBuffer struct {
	mutex sync.Mutex
	buf   bytes.Buffer
}

func (crw *concurrentBuffer) Read(p []byte) (int, error) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()
	return crw.buf.Read(p)
}

func (crw *concurrentBuffer) Write(b []byte) (int, error) {
	crw.mutex.Lock()
	defer crw.mutex.Unlock()
	return crw.buf.Write(b)
}
