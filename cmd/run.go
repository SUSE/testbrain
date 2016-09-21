// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/hpcloud/termui"
	"github.com/hpcloud/termui/termpassword"
	"github.com/spf13/cobra"

	"github.com/hpcloud/test-brain/lib"
)

// Flags from the command line are set in these variables
var testFolder string
var timeout int

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs all tests",
	Long:  `A longer description`,
	Run: func(cmd *cobra.Command, args []string) {
		ui := termui.New(os.Stdin, lib.Writer, termpassword.NewReader())

		// Open and read folder
		fileList, err := ioutil.ReadDir(testFolder)
		if err != nil {
			ui.Println(color.RedString("Could not open test folder: " + testFolder))
			termui.PrintAndExit(ui, err)
		}
		testFileList := []string{}
		for _, file := range fileList {
			if strings.HasSuffix(file.Name(), ".sh") {
				testFileList = append(testFileList, file.Name())
			}
		}
		ui.Printf("Found %d test files \n", len(testFileList))

		// Run tests
		testResults := []lib.TestResult{}
		for i, testFile := range testFileList {
			ui.Printf("Running test %s (%d/%d)\n", testFile, i+1, len(testFileList))
			result := runSingleTest(testFile)
			testResults = append(testResults, *result)
			if result.Success {
				ui.Println(color.GreenString("OK"))
			} else {
				ui.Println(color.New(color.FgRed, color.Bold).SprintfFunc()("FAILED"))
			}
		}

		// Show results
		failedTestResults := []lib.TestResult{}
		for _, result := range testResults {
			if !result.Success {
				failedTestResults = append(failedTestResults, result)
			}
		}
		ui.Printf("Tests complete: %d Passed, %d Failed\n", len(testResults)-len(failedTestResults), len(failedTestResults))
		for _, failedResult := range failedTestResults {
			redBoldColor := color.New(color.FgRed, color.Bold).SprintfFunc()
			failureName := redBoldColor("%s: Failed with code %d", failedResult.TestFile, failedResult.ExitCode)
			ui.Println(failureName)
			failureString := fmt.Sprintf("Output:\n%s", failedResult.Output)
			failureString = color.RedString(failureString)
			ui.Println(failureString)
		}
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().StringVar(&testFolder, "testfolder", "tests", "Folder containing the test files to run")
	runCmd.PersistentFlags().IntVar(&timeout, "timeout", 300, "Timeout (in seconds) for each individual test")
}

func runSingleTest(testFile string) *lib.TestResult {
	command := exec.Command(path.Join(testFolder, testFile))
	var commandOutput bytes.Buffer
	command.Stdout = &commandOutput
	command.Stderr = &commandOutput
	err := command.Start()
	if err != nil {
		return lib.ErrorTestResult(testFile, err)
	}
	timer := time.AfterFunc(time.Duration(timeout)*time.Second, func() {
		command.Process.Kill()
		commandOutput.WriteString(fmt.Sprintf("Timed out after %d seconds", timeout))
	})
	err = command.Wait()
	timer.Stop()
	exitCode := -1
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			// The program has exited with an ExitError, we can get the error code from WaitStatus
			// See https://golang.org/pkg/os/#ProcessState.Sys
			// Although the docs mention syscall.WaitStatus works on Unix, it seems to work on Windows too
			status, ok := exitErr.Sys().(syscall.WaitStatus)
			if ok {
				exitCode = status.ExitStatus()
			}
		} else {
			return lib.ErrorTestResult(testFile, err)
		}
	}

	return &lib.TestResult{
		TestFile: testFile,
		Success:  command.ProcessState.Success(),
		ExitCode: exitCode,
		Output:   string(commandOutput.Bytes()),
	}
}
