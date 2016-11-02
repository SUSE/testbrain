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

func setupFailedTestResults() []TestResult {
	failedTestResult1 := TestResult{
		TestFile: "testfile1",
		Success:  false,
		ExitCode: 1,
		Output:   "It didn't work!",
	}
	failedTestResult2 := TestResult{
		TestFile: "testfile2",
		Success:  false,
		ExitCode: 2,
		Output:   "It didn't work again!",
	}
	return []TestResult{failedTestResult1, failedTestResult2}
}

func TestGetTestScripts(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testRoot, testScripts, err := getTestScripts([]string{testFolder}, defaultInclude, defaultExclude)
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if testFolder != testRoot {
		t.Fatalf("Test root %s was not %s", testRoot, testFolder)
	}
	expected := []string{"000_script_test.sh", "001_script_test.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestGetTestScripts_IncludeFilters(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testRoot, testScripts, err := getTestScripts([]string{testFolder}, "000", defaultExclude)
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

func TestGetTestScripts_ExcludeFilters(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testRoot, testScripts, err := getTestScripts([]string{testFolder}, defaultInclude, "001")
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

	testFolder, _ := filepath.Abs("../testdata/testfolder2")
	testRoot, testScripts, err := getTestScripts([]string{testFolder}, defaultInclude, defaultExclude)
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

	testFolder, _ := filepath.Abs("../testdata/success")
	testPath := filepath.Join(testFolder, "hello_world_test.sh")
	testRoot, testScripts, err := getTestScripts([]string{testPath}, defaultInclude, defaultExclude)
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

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	testRoot, testScripts, err := getTestScripts(testFiles, "000", defaultExclude)
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

	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testFiles := []string{
		filepath.Join(testFolder, "000_script_test.sh"),
		filepath.Join(testFolder, "001_script_test.sh"),
	}
	testRoot, testScripts, err := getTestScripts(testFiles, defaultInclude, "001")
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

	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	_, testScripts1, err := getTestScriptsWithOrder([]string{testFolder}, defaultInclude, defaultExclude, false, defaultSeed)
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	_, testScripts2, err := getTestScriptsWithOrder([]string{testFolder}, defaultInclude, defaultExclude, false, defaultSeed)
	if err != nil {
		t.Fatalf("Error getting test scripts: %s", err)
	}
	if !reflect.DeepEqual(testScripts1, testScripts2) {
		t.Fatalf("Different results using the seed %d: %v and %v\n", defaultSeed, testScripts1, testScripts2)
	}
}

func TestGetTestScriptsWithOrder_SpecificSeed(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	_, testScripts, err := getTestScriptsWithOrder([]string{testFolder}, defaultInclude, defaultExclude, false, 42)
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

	testFolder, _ := filepath.Abs("../testdata/testfolder3-many-tests")
	_, testScripts, err := getTestScriptsWithOrder([]string{testFolder}, defaultInclude, defaultExclude, true, defaultSeed)
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

	testFiles := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	originalOrder := make([]string, len(testFiles))
	copy(originalOrder, testFiles)
	shuffleOrder(testFiles, defaultSeed)
	if reflect.DeepEqual(testFiles, originalOrder) {
		t.Fatalf("Shuffle order did not change: %v\n", testFiles)
	}
}

func TestRunSingleTestSuccess(t *testing.T) {
	t.Parallel()

	testFolder, _ := filepath.Abs("../testdata/success")
	testResult := runSingleTest("hello_world_test.sh", testFolder, defaultTimeout)
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

	testFolder, _ := filepath.Abs("../testdata/failure")
	testResult := runSingleTest("failure_test.sh", testFolder, defaultTimeout)
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

	testFolder, _ := filepath.Abs("../testdata")
	testResult := runSingleTest("timeout_test.sh", testFolder, 1*time.Second)
	expectedOutput := "Stuck in an infinite loop!\n" +
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

func TestOutputResults(t *testing.T) {
	_, out := setupTestUI()
	failedTestResults := setupFailedTestResults()
	outputResults(failedTestResults, 5, false, 42)
	expected := redBoldString("testfile1: Failed with exit code 1\n") +
		redBoldString("testfile2: Failed with exit code 2\n") +
		redBoldString("\nTests complete: 3 Passed, 2 Failed\n") +
		"Seed used: 42\n"
	if got := out.String(); got != expected {
		t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", expected, got)
	}
}

func TestPrintVerboseSingleTestResult(t *testing.T) {
	_, out := setupTestUI()

	type testInfo struct {
		success  bool
		output   bool
		expected string
	}
	testData := []testInfo{
		{
			success:  true,
			output:   false,
			expected: "",
		},
		{
			success:  true,
			output:   true,
			expected: color.GreenString("OK\n"),
		},
		{
			success:  false,
			output:   false,
			expected: "",
		},
		{
			success: false,
			output:  true,
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
		printVerboseSingleTestResult(result, sample.output)
		if got := out.String(); got != sample.expected {
			t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", sample.expected, got)
		}
	}
}

func TestOutputResultsJSON(t *testing.T) {
	_, out := setupTestUI()
	failedTestResults := setupFailedTestResults()
	outputResultsJSON(failedTestResults, 5, false, 42)
	expected := `{"passed":3,"failed":2,"seed":42,"inOrder":false,"failedList":[` +
		`{"filename":"testfile1","success":false,"exitcode":1,"output":"It didn't work!"},` +
		`{"filename":"testfile2","success":false,"exitcode":2,"output":"It didn't work again!"}]}`
	if got := out.String(); got != expected {
		t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", expected, got)
	}
}

func TestRunCommandSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	runner := NewRunner(false)

	err := runner.RunCommand([]string{testFolder}, defaultInclude, defaultExclude, defaultTimeout, true, false, defaultSeed, false)
	if err != nil {
		t.Fatalf("Didn't expect an error, got '%s'", err)
	}
}

func TestRunCommandFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	runner := NewRunner(false)

	err := runner.RunCommand([]string{testFolder}, defaultInclude, defaultExclude, defaultTimeout, true, false, defaultSeed, false)
	if err == nil {
		t.Fatal("Expected to get an error, got 'nil'")
	}
}
