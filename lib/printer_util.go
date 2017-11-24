package lib

import (
	"os"

	"github.com/SUSE/termui"
	"github.com/SUSE/termui/termpassword"
	"github.com/fatih/color"
)

var (
	// UI is the global printer
	UI *termui.UI
	// Green is a convenient color helper
	Green = color.New(color.FgGreen)
	// GreenBold is a convenient color helper
	GreenBold = color.New(color.FgGreen, color.Bold)
	// Red is a convenient color helper
	Red = color.New(color.FgRed)
	// RedBold is a convenient color helper
	RedBold = color.New(color.FgRed, color.Bold)
)

func init() {
	UI = termui.New(os.Stdin, Writer, termpassword.NewReader())
	// This lets us use the standard Print functions of the color library while printing to the UI
	color.Output = UI
}
