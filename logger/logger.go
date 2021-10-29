package logger

import (
	"fmt"
	"log"
)

type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// defautlLogger implements the logger contract
type defautlLogger struct {
	verbose bool
}

// New return a default implementation of custom logger
func New(verbose bool) Logger {
	return defautlLogger{verbose: verbose}
}

func (d defautlLogger) Println(v ...interface{}) {
	if d.verbose {
		log.Println("dl:", fmt.Sprintln(v...))
	}
}

func (d defautlLogger) Printf(format string, v ...interface{}) {
	if d.verbose {
		log.Println("dl:", fmt.Sprintf(format, v...))
	}
}
