// Package main provides the application's logging helpers.
//
// Set LOG_LEVEL to DEBUG, INFO (default), WARN, or ERROR to choose the
// minimum message severity. Higher-severity messages are always included.
// This file is compiled automatically with the other .go files in package
// main; it does not need to be imported from main.go.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type appLogLevel int

const (
	levelDebug appLogLevel = iota
	levelInfo
	levelWarn
	levelError
)

var (
	appLogMu         sync.RWMutex
	appLogLevelValue = parseAppLogLevel(os.Getenv("LOG_LEVEL"))
	appLogger        = log.New(os.Stdout, "", log.LstdFlags)
)

// parseAppLogLevel reads a severity name. Unknown or empty values default to INFO.
func parseAppLogLevel(value string) appLogLevel {
	switch strings.ToUpper(strings.Trim(strings.TrimSpace(value), "\"'")) {
	case "DEBUG":
		return levelDebug
	case "WARN", "WARNING":
		return levelWarn
	case "ERROR":
		return levelError
	case "", "INFO":
		return levelInfo
	default:
		return levelInfo
	}
}

// SetLogLevel changes the minimum severity at runtime. Usually LOG_LEVEL is enough.
func SetLogLevel(value string) {
	appLogMu.Lock()
	appLogLevelValue = parseAppLogLevel(value)
	appLogMu.Unlock()
}

func appLogf(level appLogLevel, label, format string, args ...any) {
	appLogMu.RLock()
	enabled := level >= appLogLevelValue
	appLogMu.RUnlock()
	if !enabled {
		return
	}
	appLogger.Printf("[%s] %s", label, fmt.Sprintf(format, args...))
}

// LogDebugf writes detailed diagnostic messages; shown with LOG_LEVEL=DEBUG.
func LogDebugf(format string, args ...any) { appLogf(levelDebug, "DEBUG", format, args...) }

// LogInfof writes routine application information. This is the default level.
func LogInfof(format string, args ...any) { appLogf(levelInfo, "INFO", format, args...) }

// LogWarnf writes warnings about conditions that may need attention.
func LogWarnf(format string, args ...any) { appLogf(levelWarn, "WARN", format, args...) }

// LogErrorf writes errors about operations that could not be completed.
func LogErrorf(format string, args ...any) { appLogf(levelError, "ERROR", format, args...) }

// LogFatalf logs a fatal error and exits with status code 1.
// Use this when the application cannot continue, for example if the HTTP
// server fails to start.
func LogFatalf(format string, args ...any) {
	appLogf(levelError, "FATAL", format, args...)
	os.Exit(1)
}
