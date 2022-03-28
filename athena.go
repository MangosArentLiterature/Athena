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
	"log"
	"os"
	"path"

	"github.com/MangosArentLiterature/Athena/internal/athena"
	"github.com/MangosArentLiterature/Athena/internal/settings"
)

var verboseFlag = flag.Bool("v", false, "toggle verbose output")
var configFlag = flag.String("c", "", "path to config directory")

func main() {
	flag.Parse()
	if *configFlag != "" {
		settings.ConfigPath = path.Clean(*configFlag)
	} else { // Get config path relative to the executable
		exe, _ := os.Executable()
		settings.ConfigPath = path.Dir(exe) + "/config"
	}
	config, err := settings.GetConfig()
	if err != nil {
		log.Fatalf("athena: failed to read config: %v", err)
	}
	athena.InitServer(config, *verboseFlag)
	athena.ListenTCP()
}
