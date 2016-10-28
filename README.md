# testbrain: HCF Acceptance Test Brain

Simple test runner. Runs all bash tests in the designated test folder, gathering results and outputs and summarizing it.

Currently, only one command is available: 

### `testbrain run`
Runs all tests in the test folder.


Flags:
```
      --json                Output in JSON format
      --testfolder string   Folder containing the test files to run (default "tests")
      --timeout int         Timeout (in seconds) for each individual test (default 300)
  -v, --verbose             Output the progress of running tests
```
