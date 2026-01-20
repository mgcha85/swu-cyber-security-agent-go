package logger

import (
	"fmt"
	"os"
	"time"
)

func Log(category, format string, v ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, category, msg)

	// Print to console
	fmt.Print(logLine)

	// Write to file
	f, err := os.OpenFile("pipeline.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(logLine)
	}
}

func Infof(category, format string, v ...interface{}) {
	Log(category, format, v...)
}

func Retrieval(format string, v ...interface{}) {
	Log("RETRIEVAL", format, v...)
}

func Agent(name, format string, v ...interface{}) {
	Log("AGENT:"+name, format, v...)
}

func SuperAgent(format string, v ...interface{}) {
	Log("SUPER_AGENT", format, v...)
}

func VLM(format string, v ...interface{}) {
	Log("VLM", format, v...)
}

func Pipeline(format string, v ...interface{}) {
	Log("PIPELINE", format, v...)
}
