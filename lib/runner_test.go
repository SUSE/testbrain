package lib

import (
	"bytes"
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

func setupDefaultRunner() (*Runner, io.ReadWriter, io.ReadWriter) {
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
	var stdout concurrentBuffer
	var stderr concurrentBuffer
	runner := NewRunner(
		&stdout,
		&stderr,
		options,
	)
	return runner, &stdout, &stderr
}

func setupTestResults() []TestResult {
	return append(setupPassedTestResults(), setupFailedTestResults()...)
}

func setupPassedTestResults() []TestResult {
	passedTestResult1 := TestResult{
		TestFile: "testfile-success-1",
		Success:  true,
		ExitCode: 0,
	}
	passedTestResult2 := TestResult{
		TestFile: "testfile-success-2",
		Success:  true,
		ExitCode: 0,
	}
	return []TestResult{passedTestResult1, passedTestResult2}
}

func setupFailedTestResults() []TestResult {
	failedTestResult1 := TestResult{
		TestFile: "testfile-failure-1",
		Success:  false,
		ExitCode: 1,
	}
	failedTestResult2 := TestResult{
		TestFile: "testfile-failure-2",
		Success:  false,
		ExitCode: 2,
	}
	return []TestResult{failedTestResult1, failedTestResult2}
}

func TestGetTestScripts(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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
	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
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

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/success")
	testFile := "hello_world_test.sh"
	testResult := r.runSingleTest(testFile, testFolder)
	if testResult.TestFile != testFile {
		t.Errorf("\nExpected TestFile: %v\nHave: %v\n", testFile, testResult.TestFile)
	}
	if testResult.Success != true {
		t.Errorf("\nExpected Success: %v\nHave: %v\n", true, testResult.Success)
	}
	if testResult.ExitCode != 0 {
		t.Errorf("\nExpected ExitCode: %v\nHave: %v\n", 0, testResult.ExitCode)
	}
}

func TestRunSingleTestFailure(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/failure")
	testFile := "failure_test.sh"
	testResult := r.runSingleTest(testFile, testFolder)
	if testResult.TestFile != testFile {
		t.Errorf("\nExpected TestFile: %v\nHave: %v\n", testFile, testResult.TestFile)
	}
	if testResult.Success != false {
		t.Errorf("\nExpected Success: %v\nHave: %v\n", false, testResult.Success)
	}
	if testResult.ExitCode != 42 {
		t.Errorf("\nExpected ExitCode: %v\nHave: %v\n", 42, testResult.ExitCode)
	}
}

func TestRunSingleTestTimeout(t *testing.T) {
	t.Parallel()

	testFile := "timeout_test.sh"

	tests := []struct {
		title              string
		verbose            bool
		expectedTestResult TestResult
		expectedStdout     string
		expectedStderr     string
	}{
		{
			title:   "verbose",
			verbose: true,
			expectedTestResult: TestResult{
				TestFile: testFile,
				Success:  false,
				ExitCode: unknownExitCode,
			},
			expectedStdout: "Timeout = 1\n" +
				"Long running process...\n",
			expectedStderr: "Killed by testbrain: Timed out after 1s\n",
		},
		{
			title:   "non-verbose",
			verbose: false,
			expectedTestResult: TestResult{
				TestFile: testFile,
				Success:  false,
				ExitCode: unknownExitCode,
			},
			expectedStdout: "",
			expectedStderr: "Killed by testbrain: Timed out after 1s\n",
		},
	}

	testFolder, _ := filepath.Abs("../testdata")

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			r, stdout, stderr := setupDefaultRunner()
			r.options.Timeout = 1 * time.Second
			r.options.Verbose = tt.verbose

			testResult := r.runSingleTest(testFile, testFolder)
			if testResult.TestFile != tt.expectedTestResult.TestFile {
				t.Errorf(
					"\nExpected TestFile: %v\nHave: %v\n",
					tt.expectedTestResult.TestFile, testResult.TestFile)
			}
			if testResult.Success != tt.expectedTestResult.Success {
				t.Errorf("\nExpected Success: %v\nHave: %v\n",
					tt.expectedTestResult.Success, testResult.Success)
			}
			if testResult.ExitCode != tt.expectedTestResult.ExitCode {
				t.Errorf("\nExpected ExitCode: %v\nHave: %v\n",
					tt.expectedTestResult.ExitCode, testResult.ExitCode)
			}

			stdoutBytes, err := ioutil.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(stderr)
			if err != nil {
				t.Fatal(err)
			}
			if stderrStr := string(stderrBytes); stderrStr != tt.expectedStderr {
				t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", tt.expectedStderr, stderrStr)
			}
		})
	}
}

func TestPrintVerboseSingleTestResult(t *testing.T) {
	tests := []struct {
		title          string
		success        bool
		json           bool
		verbose        bool
		resultOutput   io.Reader
		expectedStdout string
		expectedStderr string
	}{
		{
			title:          "Case #1",
			success:        true,
			json:           false,
			verbose:        false,
			resultOutput:   bytes.NewBufferString("something in the output"),
			expectedStdout: color.GreenString("PASSED: \n"),
			expectedStderr: "",
		},
		{
			title:          "Case #2",
			success:        true,
			json:           false,
			verbose:        true,
			resultOutput:   bytes.NewBufferString("something in the output"),
			expectedStdout: color.GreenString("PASSED: \n"),
			expectedStderr: "",
		},
		{
			title:          "Case #3",
			success:        false,
			json:           true,
			verbose:        false,
			resultOutput:   bytes.NewBufferString("something in the output"),
			expectedStdout: "",
			expectedStderr: "",
		},
		{
			title:          "Case #4",
			success:        false,
			json:           false,
			verbose:        true,
			resultOutput:   bytes.NewBufferString("something in the output"),
			expectedStdout: "",
			expectedStderr: color.New(color.FgRed, color.Bold).SprintfFunc()("FAILED: \n"),
		},
		{
			title:          "Case #5",
			success:        false,
			json:           false,
			verbose:        false,
			resultOutput:   bytes.NewBufferString("something in the output"),
			expectedStdout: "",
			expectedStderr: color.New(color.FgRed, color.Bold).SprintfFunc()(
				"FAILED: \n" +
					"Test output:\n" +
					"something in the output"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := TestResult{
				Success: tt.success,
				Output:  tt.resultOutput,
			}
			r, stdout, stderr := setupDefaultRunner()
			r.options.JSONOutput = tt.json
			r.options.Verbose = tt.verbose
			r.printVerboseSingleTestResult(result)
			stdoutBytes, err := ioutil.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(stderr)
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
	r, stdout, stderr := setupDefaultRunner()
	r.options.RandomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResults()
	expectedStdout := "Seed used: 42\n"
	expectedStderr := redBoldString("testfile-failure-1: Failed with exit code 1\n") +
		redBoldString("testfile-failure-2: Failed with exit code 2\n") +
		redBoldString("\nTests complete: 2 Passed, 0 Skipped, 2 Failed\n")
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", expectedStderr, stderrStr)
	}
}

func TestOutputResultsJSON(t *testing.T) {
	r, stdout, stderr := setupDefaultRunner()
	r.failedTestResults = setupFailedTestResults()
	r.options.RandomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResultsJSON()
	expectedStdout := `{"passed":2,"failed":2,"seed":42,"inOrder":false,"failedList":[` +
		`{"filename":"testfile-failure-1","success":false,"skipped":false,"exitcode":1},` +
		`{"filename":"testfile-failure-2","success":false,"skipped":false,"exitcode":2}]}` +
		"\n"
	expectedStderr := ""
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Errorf("\nExpected stderr:\n%q\n\nHave:\n%q\n", expectedStderr, stderrStr)
	}
}

func TestRunCommandSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	r, _, _ := setupDefaultRunner()
	r.options.TestTargets = []string{testFolder}

	err := r.RunCommand()
	if err != nil {
		t.Errorf("Didn't expect an error, got '%s'", err)
	}
}

func TestRunCommandSkip(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/skip")

	tests := []struct {
		title          string
		verbose        bool
		expectedStdout string
		expectedStderr string
	}{
		{
			title:   "Verbose",
			verbose: true,
			expectedStdout: "Found 1 test files\n" +
				"Using seed: 1552072438299530183\n" +
				"Running test skip_test.sh (1/1)\n" +
				"Something stdout\n" +
				"SKIPPED: skip_test.sh\n\n" +
				"Tests complete: 0 Passed, 1 Skipped, 0 Failed\n" +
				"Seed used: 1552072438299530183\n",
			expectedStderr: "Something stderr\n",
		},
		{
			title:   "Non-verbose",
			verbose: false,
			expectedStdout: "Found 1 test files\n" +
				"Using seed: 1552072438299530183\n" +
				"Running test skip_test.sh (1/1)\n" +
				"SKIPPED: skip_test.sh\n\n" +
				"Tests complete: 0 Passed, 1 Skipped, 0 Failed\n" +
				"Seed used: 1552072438299530183\n",
			expectedStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			r, stdout, stderr := setupDefaultRunner()
			r.options.RandomSeed = 1552072438299530183
			r.options.Verbose = tt.verbose
			r.options.TestTargets = []string{testFolder}
			err := r.RunCommand()
			if err != nil {
				t.Errorf("Didn't expect an error, got '%s'", err)
			}
			stdoutBytes, err := ioutil.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Errorf("\nExpected stdout:\n%q\n\nHave:\n%q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(stderr)
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
	r, _, _ := setupDefaultRunner()
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
