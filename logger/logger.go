package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

type LogLevel string

const (
	INFO  LogLevel = "INFO"
	DEBUG LogLevel = "DEBUG"
	ERROR LogLevel = "ERROR"
)

type ErrorInfo struct {
	Msg   string
	Stack string
}

type LogInfo struct {
	Timestamp string
	Level     LogLevel
	Service   string
	Action    string
	Message   string
	Hostname  string
	Error     *ErrorInfo
}

func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return host
}

func Log(level LogLevel, service, action, message string, errObj error) {
	entry := LogInfo{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   service,
		Action:    action,
		Message:   message,
		Hostname:  getHostname(),
	}

	if level == ERROR && errObj != nil {
		entry.Error = &ErrorInfo{
			Msg:   errObj.Error(),
			Stack: string(debug.Stack()),
		}
	}

	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}
