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
	ReportPath   string
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

func LogDebug(s string) {
	log(Debug, s)
}

func LogDebugf(format string, v ...interface{}) {
	log(Debug, fmt.Sprintf(format, v...))
}

func LogInfo(s string) {
	log(Info, s)
}

func LogInfof(format string, v ...interface{}) {
	log(Info, fmt.Sprintf(format, v...))
}

func LogWarning(s string) {
	log(Warning, s)
}

func LogWarningf(format string, v ...interface{}) {
	log(Warning, fmt.Sprintf(format, v...))
}

func LogError(s string) {
	log(Error, s)
}

func LogErrorf(format string, v ...interface{}) {
	log(Error, fmt.Sprintf(format, v...))
}

func LogFatal(s string) {
	log(Fatal, s)
}

func LogFatalf(format string, v ...interface{}) {
	log(Fatal, fmt.Sprintf(format, v...))
}

// WriteReport flushes a given area buffer to a report file.
func WriteReport(name string, buffer []string) {
	fileLock.Lock()
	err := os.WriteFile(fmt.Sprintf("report-%v-%v.log", time.Now().UTC().Format("2006-01-02T150405Z"), name), []byte(strings.Join(buffer, "\n")), 0755)
	if err != nil {
		LogError(err.Error())
	}
	fileLock.Unlock()
}
