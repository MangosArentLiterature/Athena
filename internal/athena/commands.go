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
	"flag"
	"fmt"

	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
)

var commands = map[string]func(client *Client, args []string){
	"help":  cmdHelp,
	"login": cmdLogin,
	"mkusr": cmdMakeUser,
}

type cmdPermValue struct {
	Permission uint64
	Desc       string
}

var commandperms = map[string]cmdPermValue{
	"/help":  {permissions.PermissionField["NONE"], "List all valid commands."},
	"/login": {permissions.PermissionField["NONE"], "Log in as moderator."},
	"/mkusr": {permissions.PermissionField["ADMIN"], "Creates a new moderator user."},
}

// ParseCommand calls the appropriate function for a given command.
func ParseCommand(client *Client, command string, args []string) {
	cmd := commands[command]
	if cmd != nil {
		cmd(client, args)
	} else {
		client.sendServerMessage("Invalid command.")
	}
}

// Handles /help.
func cmdHelp(client *Client, args []string) {
	s := "Recognized commands:"
	for name, attr := range commandperms {
		if permissions.HasPermission(client.perms, attr.Permission) || (attr.Permission == permissions.PermissionField["CM"] && client.area.HasCM(client.uid)) {
			s += fmt.Sprintf("\n%v: %v", name, attr.Desc)
		}
	}
	client.sendServerMessage(s)
}

// Handles /login.
func cmdLogin(client *Client, args []string) {
	usage := "usage: /login <username> <password>"
	flags := flag.NewFlagSet("", 0)
	err := flags.Parse(args)
	if err == flag.ErrHelp {
		client.sendServerMessage(usage)
		return
	}
	if len(flags.Args()) < 2 {
		client.sendServerMessage("not enough arguments\n" + usage)
		return
	}
	user := flags.Arg(0)
	pass := flags.Arg(1)
	auth, perms := db.AuthenticateUser(user, []byte(pass))
	if auth {
		client.authenticated = true
		client.perms = perms
		client.mod_name = user
		client.sendServerMessage("Logged in as moderator.")
		client.write("AUTH#1#%")
		client.sendServerMessage(fmt.Sprintf("Welcome, %v.", user))
		return
	}
	client.write("AUTH#0#%")
}

// Handles /mkusr.
func cmdMakeUser(client *Client, args []string) {
	if !permissions.HasPermission(client.perms, permissions.PermissionField["ADMIN"]) {
		client.sendServerMessage("You do not have permission to use this command.")
		return
	}

	usage := "usage: /mkusr <username> <password> <role>"
	flags := flag.NewFlagSet("", 0)
	err := flags.Parse(args)
	if err == flag.ErrHelp {
		client.sendServerMessage(usage)
		return
	}
	if len(flags.Args()) < 3 {
		client.sendServerMessage("not enough arguments\n" + usage)
		return
	}
	user := flags.Arg(0)
	pass := flags.Arg(1)
	role, err := getRole(flags.Arg(2))
	if err != nil {
		client.sendServerMessage("Invalid role.")
		return
	}
	err = db.CreateUser(user, []byte(pass), role.GetPermissions())
	if err != nil {
		client.sendServerMessage("Invalid username/password.")
		return
	}
	client.sendServerMessage("User created.")
}
