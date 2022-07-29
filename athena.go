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

package main

import (
	"flag"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/MangosArentLiterature/Athena/internal/athena"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/settings"
)

var (
	configFlag = flag.String("c", "", "path to config directory")
	reportFlag = flag.String("r", "", "path to report directory")
	logFlag    = flag.String("l", "info", "log level to use")
)

func main() {
	flag.Parse()
	if *configFlag != "" {
		settings.ConfigPath = path.Clean(*configFlag)
	} else { // Get config path relative to the executable
		exe, _ := os.Executable()
		settings.ConfigPath = path.Dir(exe) + "/config"
	}
	if *reportFlag != "" {
		logger.ReportPath = path.Clean(*reportFlag)
	} else {
		exe, _ := os.Executable()
		logger.ReportPath = path.Dir(exe) + "/reports"
	}
	switch strings.ToLower(*logFlag) {
	case "debug", "d":
		logger.CurrentLevel = logger.Debug
	case "info", "i":
		logger.CurrentLevel = logger.Info
	case "warning", "warn", "w":
		logger.CurrentLevel = logger.Warning
	case "error", "e":
		logger.CurrentLevel = logger.Error
	}

	db.DBPath = settings.ConfigPath + "/athena.db"
	config, err := settings.GetConfig()
	if err != nil {
		logger.LogFatalf("Failed to read config: %v", err)
		os.Exit(1)
	}
	err = athena.InitServer(config)
	if err != nil {
		logger.LogFatalf("Failed to initalize server: %v", err)
		athena.CleanupServer()
		os.Exit(1)
	}

	go athena.ListenTCP()

	stop := make(chan (os.Signal), 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stop:
		break
	case err := <-athena.FatalError:
		logger.LogFatal(err.Error())
		break
	}
	athena.CleanupServer()
}
