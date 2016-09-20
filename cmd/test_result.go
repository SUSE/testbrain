package cmd

type TestResult struct {
	TestFile string `json:"filename"`
	ExitCode int `json:"exitcode"`
	Output string `json:"output,omitempty"`
}