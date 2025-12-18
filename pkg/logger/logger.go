package logger

import (
	"fmt"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
)

func timestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %sINFO%s  %s\n", colorGray, timestamp(), colorReset, colorBlue, colorReset, msg)
}

func Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %s✓%s     %s\n", colorGray, timestamp(), colorReset, colorGreen, colorReset, msg)
}

func Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %sWARN%s  %s\n", colorGray, timestamp(), colorReset, colorYellow, colorReset, msg)
}

func Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %sERROR%s %s\n", colorGray, timestamp(), colorReset, colorRed, colorReset, msg)
}

func Debug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %sDEBUG%s %s\n", colorGray, timestamp(), colorReset, colorPurple, colorReset, msg)
}

func Worker(name, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s[%s]%s %s[%s]%s %s\n", colorGray, timestamp(), colorReset, colorCyan, name, colorReset, msg)
}
