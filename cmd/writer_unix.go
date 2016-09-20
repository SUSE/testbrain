// +build !windows

package cmd

import "os"

//Writer represents a unix writer
var Writer = os.Stdout