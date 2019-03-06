package lib

import (
	"github.com/fatih/color"
)

var (
	// Green is a convenient color helper
	Green = color.New(color.FgGreen).SprintfFunc()
	// GreenBold is a convenient color helper
	GreenBold = color.New(color.FgGreen, color.Bold).SprintfFunc()
	// Red is a convenient color helper
	Red = color.New(color.FgRed).SprintfFunc()
	// RedBold is a convenient color helper
	RedBold = color.New(color.FgRed, color.Bold).SprintfFunc()
)
