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

package athena

import (
	"bufio"
	"os"
	"strings"

	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
)

// ListenInput listens for input on stdin, parsing any commands.
func ListenInput() {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		cmd := strings.Split(input.Text(), " ")
		switch cmd[0] {
		case "help":
			logger.LogInfo("Recognized commands: help, mkusr, rmusr, players, getlog, say.")
		case "mkusr":
			if len(cmd) < 4 {
				logger.LogInfo("Not enough arguments for command mkusr. Usage: mkusr <username> <password> <role>.")
				break
			}
			user := cmd[1]
			pass := cmd[2]
			role, err := getRole(cmd[3])
			if err != nil {
				logger.LogInfo("Invalid role.")
				break
			}

			err = db.CreateUser(user, []byte(pass), role.GetPermissions())
			if err != nil {
				logger.LogInfof("Failed to create user: %v.", err.Error())
				break
			}
			logger.LogInfof("Sucessfully created user %v.", user)
		case "rmusr":
			if len(cmd) < 2 {
				logger.LogInfo("Not enough arguments for command rmusr. Usage: rmusr <username>.")
				break
			}
			err := db.RemoveUser(cmd[1])
			if err != nil {
				logger.LogInfof("Failed to remove user: %v.", err.Error())
				break
			}
			logger.LogInfof("Sucessfully removed user %v.", cmd[1])
		case "players":
			logger.LogInfof("There are currently %v/%v players online.", players.GetPlayerCount(), config.MaxPlayers)
		case "getlog":
			if len(cmd) < 2 {
				logger.LogInfo("Not enough arguments for command getlog. Usage: getlog <area>.")
				break
			}
			for _, a := range areas {
				if a.Name == cmd[1] {
					logger.LogInfo(strings.Join(a.GetBuffer(), "\n"))
				}
			}
		case "say":
			if len(cmd) < 2 {
				logger.LogInfo("Not enough arguments for command say. Usage: say <message>.")
				break
			}
			for c := range clients.GetClients() {
				c.sendServerMessage(cmd[1])
			}
		default:
			logger.LogInfo("Unrecognized command")
		}
	}
}
