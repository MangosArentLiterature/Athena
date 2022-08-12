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
	"sort"
	"strconv"
	"strings"

	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
)

type cmdParamList struct {
	list *[]string
}

func (v cmdParamList) String() string {
	if v.list != nil {
		return strings.Join(*v.list, ",")
	}
	return ""
}

func (v cmdParamList) Set(s string) error {
	x := strings.Split(s, ",")
	*v.list = x
	return nil
}

type cmdMapValue struct {
	Usage      string
	Desc       string
	Permission uint64
	Func       func(client *Client, args []string, usage string)
}

var commands = map[string]cmdMapValue{
	"about":    {"Usage: /about", "Prints Athena version information.", permissions.PermissionField["NONE"], cmdAbout},
	"login":    {"Usage: /login <username> <password>", "Logs in as moderator.", permissions.PermissionField["NONE"], cmdLogin},
	"logout":   {"Usage: /logout", "Logs out as moderator.", permissions.PermissionField["NONE"], cmdLogout},
	"mkusr":    {"Usage: /mkusr <username> <password> <role>", "Creates a new moderator user.", permissions.PermissionField["ADMIN"], cmdMakeUser},
	"rmusr":    {"Usage: /rmusr <username>", "Removes a moderator user.", permissions.PermissionField["ADMIN"], cmdRemoveUser},
	"setrole":  {"Usage: /setrole <username> <role>", "Updates a moderator user's role.", permissions.PermissionField["ADMIN"], cmdChangeRole},
	"kick":     {"Usage: /kick -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... <reason>", "Kicks user(s) from the server.", permissions.PermissionField["KICK"], cmdKick},
	"kickarea": {"Usage: /kickarea <uid1>,<uid2>...", "Kicks user(s) from the area.", permissions.PermissionField["CM"], cmdAreaKick},
}

// ParseCommand calls the appropriate function for a given command.
func ParseCommand(client *Client, command string, args []string) {
	if command == "help" {
		var s []string
		for name, attr := range commands {
			if permissions.HasPermission(client.Perms(), attr.Permission) || (attr.Permission == permissions.PermissionField["CM"] && client.Area().HasCM(client.Uid())) {
				s = append(s, fmt.Sprintf("/%v: %v", name, attr.Desc))
			}
		}
		sort.Strings(s)
		client.SendServerMessage("Recognized commands:\n" + strings.Join(s, "\n"))
		return
	}

	cmd := commands[command]
	if cmd.Func == nil {
		client.SendServerMessage("Invalid command.")
		return
	} else if permissions.HasPermission(client.Perms(), cmd.Permission) || (cmd.Permission == permissions.PermissionField["CM"] && client.area.HasCM(client.uid)) {
		if sliceutil.ContainsString(args, "-h") {
			client.SendServerMessage(cmd.Usage)
			return
		}
		cmd.Func(client, args, cmd.Usage)
	} else {
		client.SendServerMessage("You do not have permission to use that command.")
		return
	}
}

// Handles /login.
func cmdLogin(client *Client, args []string, usage string) {
	if client.Authenticated() {
		client.SendServerMessage("You are already logged in.")
		return
	} else if len(args) < 2 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	auth, perms := db.AuthenticateUser(args[0], []byte(args[1]))
	writeToAreaBuffer(client, "AUTH", fmt.Sprintf("Attempted login as %v.", args[0]))
	if auth {
		client.SetAuthenticated(true)
		client.SetPerms(perms)
		client.SetModName(args[0])
		client.SendServerMessage("Logged in as moderator.")
		client.Write("AUTH#1#%")
		client.SendServerMessage(fmt.Sprintf("Welcome, %v.", args[0]))
		writeToAreaBuffer(client, "AUTH", fmt.Sprintf("Logged in as %v.", args[0]))
		return
	}
	client.Write("AUTH#0#%")
	writeToAreaBuffer(client, "AUTH", fmt.Sprintf("Failed login as %v.", args[0]))
}

func cmdLogout(client *Client, _ []string, _ string) {
	if !client.Authenticated() {
		client.SendServerMessage("Invalid command.")
	}
	client.RemoveAuth()
}

// Handles /mkusr.
func cmdMakeUser(client *Client, args []string, usage string) {
	if len(args) < 3 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	role, err := getRole(args[2])
	if err != nil {
		client.SendServerMessage("Invalid role.")
		return
	}
	err = db.CreateUser(args[0], []byte(args[1]), role.GetPermissions())
	if err != nil {
		client.SendServerMessage("Invalid username/password.")
		return
	}
	client.SendServerMessage("User created.")
}

func cmdRemoveUser(client *Client, args []string, usage string) {
	if len(args) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	err := db.RemoveUser(args[0])
	if err != nil {
		client.SendServerMessage("Invalid username.")
		return
	}
	client.SendServerMessage("Removed user.")

	for c := range clients.GetClients() {
		if c.Authenticated() && c.ModName() == args[0] {
			c.RemoveAuth()
		}
	}
}

func cmdChangeRole(client *Client, args []string, usage string) {
	if len(args) < 2 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	role, err := getRole(args[1])
	if err != nil {
		client.SendServerMessage("Invalid role.")
		return
	}
	err = db.ChangePermissions(args[0], role.GetPermissions())
	if err != nil {
		client.SendServerMessage("Invalid username.")
		return
	}
	client.SendServerMessage("Role updated.")

	for c := range clients.GetClients() {
		if c.Authenticated() && c.ModName() == args[0] {
			c.SetPerms(role.GetPermissions())
		}
	}
}

func cmdKick(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	var usedList *[]string
	var useUid bool
	uids := &[]string{}
	ipids := &[]string{}
	flags.Var(&cmdParamList{uids}, "u", "")
	flags.Var(&cmdParamList{ipids}, "i", "")
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	if len(*uids) > 0 {
		useUid = true
		usedList = uids
	} else if len(*ipids) > 0 {
		usedList = ipids
	} else {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	var toKick []*Client
	for _, s := range *usedList {
		if useUid {
			x, err := strconv.Atoi(s)
			if err != nil || x == -1 {
				continue
			}
			c, err := getClientByUid(x)
			if err != nil {
				continue
			}
			toKick = append(toKick, c)
		} else {
			c := getClientsByIpid(s)
			toKick = append(toKick, c...)
		}
	}
	var count int
	for _, c := range toKick {
		c.Write(fmt.Sprintf("KK#%v#%%", strings.Join(flags.Args(), " ")))
		c.conn.Close()
		count++
	}
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
	sendPlayerArup()
}

func cmdAreaKick(client *Client, args []string, usage string) {
	if client.Area() == areas[0] {
		client.SendServerMessage("Failed to kick: Cannot kick a user from area 0.")
		return
	}
	if len(args) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	var toKick []*Client
	for _, s := range strings.Split(args[0], ",") {
		uid, err := strconv.Atoi(s)
		if err != nil || uid == -1 {
			continue
		}
		c, err := getClientByUid(uid)
		if err != nil {
			continue
		}
		toKick = append(toKick, c)
	}

	var count int
	for _, c := range toKick {
		if c.Area() != client.Area() {
			continue
		}
		if c == client {
			client.SendServerMessage("You can't kick yourself from the area.")
			continue
		}
		c.Area().RemoveChar(c.CharID())
		if !areas[0].AddChar(c.CharID()) {
			c.SetCharID(-1)
			areas[0].AddChar(-1)
			c.Write("DONE#%")
		}
		c.SetArea(areas[0])
		c.SendServerMessage("You were kicked from the area!")
		count++
	}
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
	sendPlayerArup()
}

func cmdAbout(client *Client, _ []string, _ string) {
	client.SendServerMessage(fmt.Sprintf("Running Athena version %v.\nAthena is open source software, for documentation, bug reports, and source code, see: %v",
		version, "github.com/MangosArentLiterature/Athena"))
}
