// +build !windows

package lib

import "os"

//Writer represents a unix writer
var Writer = os.Stdout
