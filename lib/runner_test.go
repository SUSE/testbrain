package lib

import (
	"io"
	"io/ioutil"
	"path/filepath"
	"reflect"
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
	testTargets := []string{}
	includeReStr := defaultInclude
	excludeReStr := defaultExclude
	timeout := defaultTimeout
	inOrder := false
	randomSeed := defaultSeed
	jsonOutput := false
	verbose := false
	dryRun := false
	options := NewRunnerOptions(
		testTargets,
		includeReStr,
		excludeReStr,
		timeout,
		inOrder,
		randomSeed,
		jsonOutput,
		verbose,
		dryRun,
	)
	stdout := NewConcurrentBuffer()
	stderr := NewConcurrentBuffer()
	runner := NewRunner(
		stdout,
		stderr,
		options,
	)
	return runner, stdout, stderr
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
	r.options.testTargets = []string{testFolder}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root '%s' was not '%s'", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh", "001_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFilters(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.options.testTargets = []string{testFolder}
	r.options.includeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root '%s' was not '%s'", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersMultiFolder(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.testTargets = []string{testFolderA, testFolderB}
	r.options.includeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	expectedRoot := "testdata"
	if expectedRoot != filepath.Base(testRoot) {
		t.Fatalf("Test root %s was not %s", testRoot, expectedRoot)
	}
	expected := []string{
		"testfolder1/000_script_test.sh",
		"testfolder3-many-tests/000_script_test.sh",
	}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersNoMatch(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.options.testTargets = []string{testFolder}
	r.options.includeReStr = "002" // testfolder1 has only 000 and 001

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}

	if len(testScripts) > 0 {
		t.Fatalf("Expected: []\nHave:     %v\n", testScripts)
	}
}

func TestGetTestScripts_IncludeFiltersMultiFolderNoMatch(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.testTargets = []string{testFolderA, testFolderB}
	r.options.includeReStr = "005"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	expectedRoot := ""
	if expectedRoot != testRoot {
		t.Fatalf("Test root '%s' was not '%s'", testRoot, expectedRoot)
	}
	if len(testScripts) > 0 {
		t.Fatalf("Expected: []\nHave:     %v\n", testScripts)
	}
}

func TestGetTestScripts_ExcludeFilters(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r, _, _ := setupDefaultRunner()
	r.options.testTargets = []string{testFolder}
	r.options.excludeReStr = "001"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_NestedDirectories(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder2")
	r.options.testTargets = []string{testFolder}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"nested_directory/test_file_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_OnlyOneFile(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/success")
	testPath := filepath.Join(testFolder, "hello_world_test.sh")
	r.options.testTargets = []string{testPath}

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test script: %s", err)
	}
	if testRoot != testFolder {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"hello_world_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:      %v\n", expected, testScripts)
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
	r.options.testTargets = testFiles
	r.options.includeReStr = "000"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
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
	r.options.testTargets = testFiles
	r.options.excludeReStr = "001"

	testRoot, testScripts, err := r.getTestScripts()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScriptsWithOrder_SameSeedSameOrder(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.testTargets = []string{testFolder}

	_, testScripts1, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	_, testScripts2, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if !reflect.DeepEqual(testScripts1, testScripts2) {
		t.Fatalf("Different results using the seed %d: %v and %v\n", defaultSeed, testScripts1, testScripts2)
	}
}

func TestGetTestScriptsWithOrder_SpecificSeed(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.testTargets = []string{testFolder}
	r.options.randomSeed = 42

	_, testScripts, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	expected := []string{"001_script_test.sh", "004_script_test.sh", "002_script_test.sh", "003_script_test.sh", "000_script_test.sh"}
	if !reflect.DeepEqual(expected, testScripts) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScriptsWithOrder_InOrder(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.options.testTargets = []string{testFolder}
	r.options.inOrder = true

	_, testScripts, err := r.getTestScriptsWithOrder()
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	expected := []string{"000_script_test.sh", "001_script_test.sh", "002_script_test.sh", "003_script_test.sh", "004_script_test.sh"}
	if !reflect.DeepEqual(expected, testScripts) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
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
		t.Fatalf("Shuffle order did not change: %v\n", testFiles)
	}
}

func TestRunSingleTestSuccess(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/success")
	testResult := r.runSingleTest("hello_world_test.sh", testFolder)
	expected := TestResult{
		TestFile: "hello_world_test.sh",
		Success:  true,
		ExitCode: 0,
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
	}
}

func TestRunSingleTestFailure(t *testing.T) {
	t.Parallel()

	r, _, _ := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/failure")
	testResult := r.runSingleTest("failure_test.sh", testFolder)
	expected := TestResult{
		TestFile: "failure_test.sh",
		Success:  false,
		ExitCode: 42,
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
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
				ExitCode: -1,
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
				ExitCode: -1,
			},
			expectedStdout: "",
			expectedStderr: "Killed by testbrain: Timed out after 1s\n" +
				"Test output:\n" +
				"Timeout = 1\n" +
				"Long running process...\n",
		},
	}

	testFolder, _ := filepath.Abs("../testdata")

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			r, stdout, stderr := setupDefaultRunner()
			r.options.timeout = 1 * time.Second
			r.options.verbose = tt.verbose

			testResult := r.runSingleTest(testFile, testFolder)
			if !reflect.DeepEqual(testResult, tt.expectedTestResult) {
				t.Fatalf("Expected: %v\nHave:     %v\n", tt.expectedTestResult, testResult)
			}

			stdoutBytes, err := ioutil.ReadAll(stdout)
			if err != nil {
				t.Fatal(err)
			}
			if stdoutStr := string(stdoutBytes); stdoutStr != tt.expectedStdout {
				t.Fatalf("Expected stdout:\n %q\n\nHave:\n %q\n", tt.expectedStdout, stdoutStr)
			}
			stderrBytes, err := ioutil.ReadAll(stderr)
			if err != nil {
				t.Fatal(err)
			}
			if stderrStr := string(stderrBytes); stderrStr != tt.expectedStderr {
				t.Fatalf("Expected stderr:\n %q\n\nHave:\n %q\n", tt.expectedStderr, stderrStr)
			}
		})
	}
}

func TestPrintVerboseSingleTestResult(t *testing.T) {
	type testInfo struct {
		success        bool
		json           bool
		verbose        bool
		expectedStdout string
		expectedStderr string
	}
	testData := []testInfo{
		{
			success:        true,
			json:           false,
			verbose:        false,
			expectedStdout: "",
			expectedStderr: "",
		},
		{
			success:        true,
			json:           false,
			verbose:        true,
			expectedStdout: color.GreenString("OK\n"),
			expectedStderr: "",
		},
		{
			success:        false,
			json:           true,
			verbose:        false,
			expectedStdout: "",
			expectedStderr: "",
		},
		{
			success:        false,
			json:           false,
			verbose:        true,
			expectedStdout: "",
			expectedStderr: color.New(color.FgRed, color.Bold).SprintfFunc()("FAILED\n"),
		},
	}

	// When we move to go1.7 we can do subtests
	for _, sample := range testData {
		result := TestResult{
			Success: sample.success,
		}
		r, stdout, stderr := setupDefaultRunner()
		r.options.jsonOutput = sample.json
		r.options.verbose = sample.verbose
		r.printVerboseSingleTestResult(result)
		stdoutBytes, err := ioutil.ReadAll(stdout)
		if err != nil {
			t.Fatal(err)
		}
		if stdoutStr := string(stdoutBytes); stdoutStr != sample.expectedStdout {
			t.Fatalf("Expected stdout:\n %q\n\nHave:\n %q\n", sample.expectedStdout, stdoutStr)
		}
		stderrBytes, err := ioutil.ReadAll(stderr)
		if err != nil {
			t.Fatal(err)
		}
		if stderrStr := string(stderrBytes); stderrStr != sample.expectedStderr {
			t.Fatalf("Expected stderr:\n %q\n\nHave:\n %q\n", sample.expectedStderr, stderrStr)
		}
	}
}

func TestOutputResults(t *testing.T) {
	r, stdout, stderr := setupDefaultRunner()
	r.options.randomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResults()
	expectedStdout := "Seed used: 42\n"
	expectedStderr := redBoldString("testfile-failure-1: Failed with exit code 1\n") +
		redBoldString("testfile-failure-2: Failed with exit code 2\n") +
		redBoldString("\nTests complete: 2 Passed, 2 Failed\n")
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Fatalf("Expected stdout:\n %q\n\nHave:\n %q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Fatalf("Expected stderr:\n %q\n\nHave:\n %q\n", expectedStderr, stderrStr)
	}
}

func TestOutputResultsJSON(t *testing.T) {
	r, stdout, stderr := setupDefaultRunner()
	r.failedTestResults = setupFailedTestResults()
	r.options.randomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResultsJSON()
	expectedStdout := `{"passed":2,"failed":2,"seed":42,"inOrder":false,"failedList":[` +
		`{"filename":"testfile-failure-1","success":false,"exitcode":1},` +
		`{"filename":"testfile-failure-2","success":false,"exitcode":2}]}` +
		"\n"
	expectedStderr := ""
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		t.Fatal(err)
	}
	if stdoutStr := string(stdoutBytes); stdoutStr != expectedStdout {
		t.Fatalf("Expected stdout:\n %q\n\nHave:\n %q\n", expectedStdout, stdoutStr)
	}
	stderrBytes, err := ioutil.ReadAll(stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stderrStr := string(stderrBytes); stderrStr != expectedStderr {
		t.Fatalf("Expected stderr:\n %q\n\nHave:\n %q\n", expectedStderr, stderrStr)
	}
}

func TestRunCommandSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	r, _, _ := setupDefaultRunner()
	r.options.testTargets = []string{testFolder}

	err := r.RunCommand()
	if err != nil {
		t.Fatalf("Didn't expect an error, got '%s'", err)
	}
}

func TestRunCommandFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	r, _, _ := setupDefaultRunner()
	r.options.testTargets = []string{testFolder}

	err := r.RunCommand()
	if err == nil {
		t.Fatal("Expected to get an error, got 'nil'")
	}
}
