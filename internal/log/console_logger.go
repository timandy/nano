package log

import (
	"log"
	"os"
)

// ConsoleLogger represents  the log interface
type ConsoleLogger log.Logger

func NewConsoleLogger() *ConsoleLogger {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	return (*ConsoleLogger)(logger)
}

func (c *ConsoleLogger) Info(args ...any) {
	_ = (*log.Logger)(c).Output(2, FormatArgs(args...))
}

func (c *ConsoleLogger) Error(args ...any) {
	_ = (*log.Logger)(c).Output(2, FormatArgs(args...))
}

func (c *ConsoleLogger) Fatal(args ...any) {
	_ = (*log.Logger)(c).Output(2, FormatArgs(args...))
	os.Exit(1)
}
