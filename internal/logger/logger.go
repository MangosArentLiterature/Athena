/* Athena - A server for Attorney Online 2 written in Go
Copyright (C) 2022 MangosArentLiterature <mango@transmenace.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>. */

package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warning
	Error
	Fatal
)

var (
	levelToString = map[LogLevel]string{
		Debug:   "DEBUG",
		Info:    "INFO",
		Warning: "WARN",
		Error:   "ERROR",
		Fatal:   "FATAL",
	}
	LogPath      string
	CurrentLevel LogLevel
	outputLock   sync.Mutex
	fileLock     sync.Mutex
	DebugNetwork bool
)

// log writes a message to standard output if the level matches the server's set log level.
func log(level LogLevel, s string) {
	if level < CurrentLevel {
		return
	}
	outputLock.Lock()
	fmt.Printf("%v: %v: %v\n", time.Now().UTC().Format(time.StampMilli), levelToString[level], s)
	outputLock.Unlock()
}

// LogDebug prints a debug message to stdout. Arguments are handled in the manner of fmt.Print.
func LogDebug(s string) {
	log(Debug, s)
}

// LogDebugf prints a debug message to stdout. Arguments are handled in the manner of fmt.Printf.
func LogDebugf(format string, v ...interface{}) {
	log(Debug, fmt.Sprintf(format, v...))
}

// LogInfo prints an info message to stdout. Arguments are handled in the manner of fmt.Print.
func LogInfo(s string) {
	log(Info, s)
}

// LogInfof prints an info message to stdout. Arguments are handled in the manner of fmt.Printf.
func LogInfof(format string, v ...interface{}) {
	log(Info, fmt.Sprintf(format, v...))
}

// LogWarning prints a warning message to stdout. Arguments are handled in the manner of fmt.Print.
func LogWarning(s string) {
	log(Warning, s)
}

// LogWarningf prints a warning message to stdout. Arguments are handled in the manner of fmt.Printf.
func LogWarningf(format string, v ...interface{}) {
	log(Warning, fmt.Sprintf(format, v...))
}

// LogError prints an error message to stdout. Arguments are handled in the manner of fmt.Print.
func LogError(s string) {
	log(Error, s)
}

// LogErrorf prints an error message to stdout. Arguments are handled in the manner of fmt.Printf.
func LogErrorf(format string, v ...interface{}) {
	log(Error, fmt.Sprintf(format, v...))
}

// LogFatal prints a fatal error message to stdout. Arguments are handled in the manner of fmt.Print.
func LogFatal(s string) {
	log(Fatal, s)
}

// LogFatalf prints a fatal error message to stdout. Arguments are handled in the manner of fmt.Printf.
func LogFatalf(format string, v ...interface{}) {
	log(Fatal, fmt.Sprintf(format, v...))
}

// WriteReport flushes a given area buffer to a report file.
func WriteReport(name string, buffer []string) {
	fileLock.Lock()
	err := os.WriteFile(fmt.Sprintf("%v/report-%v-%v.log", LogPath, time.Now().UTC().Format("2006-01-02T150405Z"), name), []byte(strings.Join(buffer, "\n")), 0755)
	if err != nil {
		LogError(err.Error())
	}
	fileLock.Unlock()
}

// WriteAudit writes a line to the server's audit log.
func WriteAudit(s string) {
	fileLock.Lock()
	f, err := os.OpenFile(LogPath+"/audit.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		LogError(err.Error())
	}
	_, err = f.WriteString(s + "\n")
	if err != nil {
		LogError(err.Error())
	}
	f.Close()
	fileLock.Unlock()
}
