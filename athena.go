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
	"syscall"

	"github.com/MangosArentLiterature/Athena/internal/athena"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/settings"
)

var (
	configFlag   = flag.String("c", "config", "path to config directory")
	netDebugFlag = flag.Bool("netdebug", false, "log raw network traffic")
)

func main() {
	flag.Parse()
	if *configFlag != "" {
		settings.ConfigPath = path.Clean(*configFlag)
	}
	config, err := settings.GetConfig()
	if err != nil {
		logger.LogFatalf("failed to read config: %v", err)
		os.Exit(1)
	}
	logger.LogPath = path.Clean(config.LogDir)

	switch config.LogLevel {
	case "debug":
		logger.CurrentLevel = logger.Debug
	case "info":
		logger.CurrentLevel = logger.Info
	case "warning":
		logger.CurrentLevel = logger.Warning
	case "error":
		logger.CurrentLevel = logger.Error
	case "fatal":
		logger.CurrentLevel = logger.Fatal
	}
	logger.DebugNetwork = *netDebugFlag
	db.DBPath = settings.ConfigPath + "/athena.db"

	err = athena.InitServer(config)
	if err != nil {
		logger.LogFatalf("Failed to initalize server: %v", err)
		athena.CleanupServer()
		os.Exit(1)
	}
	logger.LogInfo("Started server.")
	go athena.ListenTCP()
	go athena.ListenInput()
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
	logger.LogInfo("Stopping server.")
}
