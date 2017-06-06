# testbrain: Acceptance Test Brain

Simple test runner. Runs all bash tests in the designated test folder, gathering results and outputs and summarizing it.

Currently, only one command is available: 

### `testbrain run`
Runs all tests in the test folder.

```
Usage:
  testbrain run [flags] [files...]

Flags:
  -n, --dry-run          Do not actually run the tests
      --exclude string   Regular expression of subset of tests to not run, applied after --include (default "^$")
      --in-order         Do not randomize test order
      --include string   Regular expression of subset of tests to run (default "_test\\.sh$")
      --json             Output in JSON format
      --seed int         Random seed used to determine the order of tests (default -1)
      --timeout int      Timeout (in seconds) for each individual test (default 300)
  -v, --verbose          Output the progress of running tests

Global Flags:
      --config string   config file (default is $HOME/.test-brain.yaml)
```
