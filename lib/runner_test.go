package lib

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/hpcloud/termui"
)

var (
	defaultTimeout = 300 * time.Second
	defaultInclude = "_test\\.sh$"
	defaultExclude = "^$"
	defaultSeed    = time.Now().UnixNano()

	redBoldString = color.New(color.FgRed, color.Bold).SprintfFunc()
)

func setupTestUI() (*bytes.Buffer, *bytes.Buffer) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}
	UI = termui.New(in, out, nil)
	color.Output = UI
	color.NoColor = false
	return in, out
}

func setupDefaultRunner() *Runner {
	return &Runner{
		TestTargets: []string{},
		IncludeRe:   defaultInclude,
		ExcludeRe:   defaultExclude,
		Timeout:     defaultTimeout,
		InOrder:     false,
		RandomSeed:  defaultSeed,
		JSONOutput:  false,
		Verbose:     false,
		DryRun:      false,
	}
}

func setupTestResults() []TestResult {
	return append(setupPassedTestResults(), setupFailedTestResults()...)
}

func setupPassedTestResults() []TestResult {
	passedTestResult1 := TestResult{
		TestFile: "testfile-success-1",
		Success:  true,
		ExitCode: 0,
		Output:   "It worked!",
	}
	passedTestResult2 := TestResult{
		TestFile: "testfile-success-2",
		Success:  true,
		ExitCode: 0,
		Output:   "It worked again!",
	}
	return []TestResult{passedTestResult1, passedTestResult2}
}

func setupFailedTestResults() []TestResult {
	failedTestResult1 := TestResult{
		TestFile: "testfile-failure-1",
		Success:  false,
		ExitCode: 1,
		Output:   "It didn't work!",
	}
	failedTestResult2 := TestResult{
		TestFile: "testfile-failure-2",
		Success:  false,
		ExitCode: 2,
		Output:   "It didn't work again!",
	}
	return []TestResult{failedTestResult1, failedTestResult2}
}

func TestGetTestScripts(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.TestTargets = []string{testFolder}

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.TestTargets = []string{testFolder}
	r.IncludeRe = "000"

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

	r := setupDefaultRunner()
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.TestTargets = []string{testFolderA, testFolderB}
	r.IncludeRe = "000"

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	r.TestTargets = []string{testFolder}
	r.IncludeRe = "002" // testfolder1 has only 000 and 001

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

	r := setupDefaultRunner()
	testFolderA, _ := filepath.Abs("../testdata/testfolder1")
	testFolderB, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.TestTargets = []string{testFolderA, testFolderB}
	r.IncludeRe = "005"

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
	r := setupDefaultRunner()
	r.TestTargets = []string{testFolder}
	r.ExcludeRe = "001"

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder2")
	r.TestTargets = []string{testFolder}

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/success")
	testPath := filepath.Join(testFolder, "hello_world_test.sh")
	r.TestTargets = []string{testPath}

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	r.TestTargets = testFiles
	r.IncludeRe = "000"

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	r.TestTargets = testFiles
	r.ExcludeRe = "001"

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.TestTargets = []string{testFolder}

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.TestTargets = []string{testFolder}
	r.RandomSeed = 42

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

	r := setupDefaultRunner()
	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	r.TestTargets = []string{testFolder}
	r.InOrder = true

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

	r := setupDefaultRunner()

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

	r := setupDefaultRunner()

	testFolder, _ := filepath.Abs("../testdata/success")
	testResult := r.runSingleTest("hello_world_test.sh", testFolder)
	expected := TestResult{
		TestFile: "hello_world_test.sh",
		Success:  true,
		ExitCode: 0,
		Output:   "Hello World!\n",
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
	}
}

func TestRunSingleTestFailure(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner()

	testFolder, _ := filepath.Abs("../testdata/failure")
	testResult := r.runSingleTest("failure_test.sh", testFolder)
	expected := TestResult{
		TestFile: "failure_test.sh",
		Success:  false,
		ExitCode: 42,
		Output:   "Goodbye World!\n",
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
	}
}

func TestRunSingleTestTimeout(t *testing.T) {
	t.Parallel()

	r := setupDefaultRunner()
	r.Timeout = 1 * time.Second

	testFolder, _ := filepath.Abs("../testdata")
	testResult := r.runSingleTest("timeout_test.sh", testFolder)
	expectedOutput := "Timeout = 1\n" +
		"Stuck in an infinite loop!\n" +
		"Stuck in an infinite loop!\n" +
		"Stuck in an infinite loop!\n" +
		"Killed by testbrain: Timed out after 1s"
	expected := TestResult{
		TestFile: "timeout_test.sh",
		Success:  false,
		ExitCode: -1,
		Output:   expectedOutput,
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
	}
}

func TestPrintVerboseSingleTestResult(t *testing.T) {
	_, out := setupTestUI()
	r := setupDefaultRunner()

	type testInfo struct {
		success  bool
		json     bool
		verbose  bool
		expected string
	}
	testData := []testInfo{
		{
			success:  true,
			json:     false,
			verbose:  false,
			expected: "",
		},
		{
			success:  true,
			json:     false,
			verbose:  true,
			expected: color.GreenString("OK\n"),
		},
		{
			success:  false,
			json:     true,
			verbose:  false,
			expected: "",
		},
		{
			success: false,
			json:    false,
			verbose: true,
			expected: color.New(color.FgRed, color.Bold).SprintfFunc()("FAILED\n") +
				color.RedString("Output:\ntest output\n"),
		},
	}

	// When we move to go1.7 we can do subtests
	for _, sample := range testData {
		result := TestResult{
			Success: sample.success,
			Output:  "test output",
		}
		out.Reset()
		r.JSONOutput = sample.json
		r.Verbose = sample.verbose
		r.printVerboseSingleTestResult(result)
		if got := out.String(); got != sample.expected {
			t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", sample.expected, got)
		}
	}
}

func TestOutputResults(t *testing.T) {
	_, out := setupTestUI()
	r := setupDefaultRunner()
	r.RandomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResults()
	expected := redBoldString("testfile-failure-1: Failed with exit code 1\n") +
		redBoldString("testfile-failure-2: Failed with exit code 2\n") +
		redBoldString("\nTests complete: 2 Passed, 2 Failed\n") +
		"Seed used: 42\n"
	if got := out.String(); got != expected {
		t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", expected, got)
	}
}

func TestOutputResultsJSON(t *testing.T) {
	_, out := setupTestUI()
	r := setupDefaultRunner()
	r.failedTestResults = setupFailedTestResults()
	r.RandomSeed = 42
	r.testResults = setupTestResults()
	r.failedTestResults = setupFailedTestResults()

	r.outputResultsJSON()
	expected := `{"passed":2,"failed":2,"seed":42,"inOrder":false,"failedList":[` +
		`{"filename":"testfile-failure-1","success":false,"exitcode":1,"output":"It didn't work!"},` +
		`{"filename":"testfile-failure-2","success":false,"exitcode":2,"output":"It didn't work again!"}]}`
	if got := out.String(); got != expected {
		t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", expected, got)
	}
}

func TestRunCommandSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	r := setupDefaultRunner()
	r.TestTargets = []string{testFolder}

	err := r.RunCommand()
	if err != nil {
		t.Fatalf("Didn't expect an error, got '%s'", err)
	}
}

func TestRunCommandFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	r := setupDefaultRunner()
	r.TestTargets = []string{testFolder}

	err := r.RunCommand()
	if err == nil {
		t.Fatal("Expected to get an error, got 'nil'")
	}
}
