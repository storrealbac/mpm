package ui

import (
	"fmt"
)

// Helper functions for printing with icons and styles

func PrintHeader(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(HeaderStyle.Render(msg))
}

func PrintTitle(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(TitleStyle.Render(msg))
}

func PrintSuccess(format string, a ...interface{}) {
	prefix := SuccessStyle.Render("SUCCESS")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s  %s\n", prefix, msg)
}

func PrintError(format string, a ...interface{}) {
	prefix := ErrorStyle.Render("ERROR")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s    %s\n", prefix, msg)
}

func PrintWarning(format string, a ...interface{}) {
	prefix := WarningStyle.Render("WARN")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s     %s\n", prefix, msg)
}

func PrintInfo(format string, a ...interface{}) {
	prefix := InfoStyle.Render("INFO")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("  %s     %s\n", prefix, msg)
}

func PrintStep(step int, total int, format string, a ...interface{}) {
	prefix := fmt.Sprintf("[%d/%d]", step, total)
	icon := InfoStyle.Render(prefix)
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", icon, msg)
}

func PrintMPM() string {
	return MPMStyle.Render("mpm")
}
