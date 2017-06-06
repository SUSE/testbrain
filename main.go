package main

import (
	"errors"
	"os"

	"github.com/suse/termui"
	"github.com/suse/testbrain/cmd"
)

// This variable is set in the make/build script, during the go build
var version = "0"

func main() {
	ui := termui.New(os.Stdin, os.Stdout, nil)

	switch {
	case version == "":
		termui.PrintAndExit(ui, errors.New("testbrain was built incorrectly and its version string is empty"))
	case version == "0":
		termui.PrintAndExit(ui, errors.New("testbrain was built incorrectly and it doesn't have a proper version string"))
	}

	cmd.Execute(version)
}
