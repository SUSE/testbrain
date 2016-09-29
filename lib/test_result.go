package lib

// TestResult contains the result of a single test script
type TestResult struct {
	TestFile string `json:"filename"`
	Success  bool   `json:"success"`
	ExitCode int    `json:"exitcode"`
	Output   string `json:"output"`
}

// UnknownExitCode represents an unknown exit code
const UnknownExitCode = -1

// ErrorTestResult returns a TestResult for tests that failed with an error (bash syntax error, etc.)
func ErrorTestResult(testFile string, err error) TestResult {
	return TestResult{
		TestFile: testFile,
		Success:  false,
		ExitCode: UnknownExitCode,
		Output:   err.Error(),
	}
}
