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
	msg   string
	stack string
}

type LogInfo struct {
	timestamp string
	level     LogLevel
	service   string
	action    string
	message   string
	hostname  string
	error     *ErrorInfo
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
		timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		level:     level,
		service:   service,
		action:    action,
		message:   message,
		hostname:  getHostname(),
	}

	if level == ERROR && errObj != nil {
		entry.error = &ErrorInfo{
			msg:   errObj.Error(),
			stack: string(debug.Stack()),
		}
	}

	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}
