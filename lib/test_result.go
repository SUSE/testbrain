package lib

import "io"

// TestResult contains the result of a single test script.
type TestResult struct {
	TestFile string    `json:"filename"`
	Success  bool      `json:"success"`
	ExitCode int       `json:"exitcode"`
	Output   io.Reader `json:"-"`
}
