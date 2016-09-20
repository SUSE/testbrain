package lib

type TestResult struct {
	TestFile string `json:"filename"`
	Success  bool   `json:"success"`
	ExitCode int    `json:"exitcode"`
	Output   string `json:"output,omitempty"`
}

// Returns a TestResult for tests that failed with an error (bash syntax error, etc.)
func ErrorTestResult(testFile string, err error) *TestResult {
	return &TestResult{
		TestFile: testFile,
		Success:  false,
		ExitCode: -1,
		Output:   err.Error(),
	}
}
