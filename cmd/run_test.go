package cmd

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/hpcloud/termui"

	"github.com/hpcloud/testbrain/lib"
)

const defaultTimeout = 300 * time.Second

func setupTestUI() (*bytes.Buffer, *bytes.Buffer) {
	in, out := &bytes.Buffer{}, &bytes.Buffer{}
	ui = termui.New(in, out, nil)
	color.Output = ui
	color.NoColor = false
	return in, out
}

func setupFailedTestResults() []lib.TestResult {
	failedTestResult1 := lib.TestResult{
		TestFile: "testfile1",
		Success:  false,
		ExitCode: 1,
		Output:   "It didn't work!",
	}
	failedTestResult2 := lib.TestResult{
		TestFile: "testfile2",
		Success:  false,
		ExitCode: 2,
		Output:   "It didn't work again!",
	}
	return []lib.TestResult{failedTestResult1, failedTestResult2}
}

func TestGetTestScripts(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/testfolder1")
	testScripts := getTestScripts(testFolder)
	expected := []string{"000_script.sh", "001_script.sh"}
	if !reflect.DeepEqual(testScripts, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testScripts)
	}
}

func TestRunSingleTestSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	testResult := runSingleTest("hello_world.sh", testFolder, defaultTimeout)
	expected := lib.TestResult{
		TestFile: "hello_world.sh",
		Success:  true,
		ExitCode: 0,
		Output:   "Hello World!\n",
	}
	if !reflect.DeepEqual(testResult, expected) {
		t.Fatalf("Expected: %v\nHave:     %v\n", expected, testResult)
	}
}

func TestRunSingleTestFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	testResult := runSingleTest("failure_test.sh", testFolder, defaultTimeout)
	expected := lib.TestResult{
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
	testFolder, _ := filepath.Abs("../testdata")
	testResult := runSingleTest("timeout.sh", testFolder, 1*time.Second)
	expectedOutput := "Stuck in an infinite loop!\nStuck in an infinite loop!\nStuck in an infinite loop!\nKilled by testbrain: Timed out after 1s"
	expected := lib.TestResult{
		TestFile: "timeout.sh",
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
	outputResults(failedTestResults, 5)
	expected := "\x1b[31;1mtestfile1: Failed with exit code 1\n\x1b[0m\x1b[31;1mtestfile2: Failed with exit code 2\n\x1b[0m\x1b[31;1m\nTests complete: 3 Passed, 2 Failed\n\x1b[0m"
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
		result := lib.TestResult{
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
	outputResultsJSON(failedTestResults, 5)
	expected := `{"passed":3,"failed":2,"failedList":[{"filename":"testfile1","success":false,"exitcode":1,"output":"It didn't work!"},{"filename":"testfile2","success":false,"exitcode":2,"output":"It didn't work again!"}]}`
	if got := out.String(); got != expected {
		t.Fatalf("Expected:\n %q\n\nHave:\n %q\n", expected, got)
	}
}

func TestRunCommandSuccess(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/success")
	err := runCommand(testFolder, defaultTimeout, true, false)
	if err != nil {
		t.Fatalf("Didn't expect an error, got '%s'", err)
	}
}

func TestRunCommandFailure(t *testing.T) {
	testFolder, _ := filepath.Abs("../testdata/failure")
	err := runCommand(testFolder, defaultTimeout, true, false)
	if err == nil {
		t.Fatal("Expected to get an error, got 'nil'")
	}
}
