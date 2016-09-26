# testbrain: HCF Acceptance Test Brain

Simple test runner. Runs all bash tests in the designated test folder, gathering results and outputs and summarizing it.

Currently, only one command is available: 

### `testbrain run`
Runs all tests in the test folder.

Flags:  
`-v`: Output the progress of running tests  
`--testfolder`: Folder containing the test files to run  
`--timeout`: Timeout in seconds for each individual test (defaults to 5 minutes)  
`--json`: Output in JSON format  
