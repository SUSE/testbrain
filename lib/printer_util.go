package lib

import (
	"os"

	"github.com/fatih/color"
	"github.com/hpcloud/termui"
	"github.com/hpcloud/termui/termpassword"
)

var (
	UI        *termui.UI
	Green     = color.New(color.FgGreen)
	GreenBold = color.New(color.FgGreen, color.Bold)
	Red       = color.New(color.FgRed)
	RedBold   = color.New(color.FgRed, color.Bold)
)

func init() {
	UI = termui.New(os.Stdin, Writer, termpassword.NewReader())
	// This lets us use the standard Print functions of the color library while printing to the UI
	color.Output = UI
}
