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
	"time"

	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
	"github.com/xhit/go-str2duration/v2"
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
	Args       int
	Usage      string
	Desc       string
	Permission uint64
	Func       func(client *Client, args []string, usage string)
}

var commands = map[string]cmdMapValue{
	"about":    {0, "Usage: /about", "Prints Athena version information.", permissions.PermissionField["NONE"], cmdAbout},
	"login":    {2, "Usage: /login <username> <password>", "Logs in as moderator.", permissions.PermissionField["NONE"], cmdLogin},
	"logout":   {1, "Usage: /logout", "Logs out as moderator.", permissions.PermissionField["NONE"], cmdLogout},
	"mkusr":    {3, "Usage: /mkusr <username> <password> <role>", "Creates a new moderator user.", permissions.PermissionField["ADMIN"], cmdMakeUser},
	"rmusr":    {1, "Usage: /rmusr <username>", "Removes a moderator user.", permissions.PermissionField["ADMIN"], cmdRemoveUser},
	"setrole":  {2, "Usage: /setrole <username> <role>", "Updates a moderator user's role.", permissions.PermissionField["ADMIN"], cmdChangeRole},
	"kick":     {3, "Usage: /kick -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... <reason>", "Kicks user(s) from the server.", permissions.PermissionField["KICK"], cmdKick},
	"kickarea": {1, "Usage: /kickarea <uid1>,<uid2>...", "Kicks user(s) from the area.", permissions.PermissionField["CM"], cmdAreaKick},
	"ban":      {3, "Usage: /ban -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... [-d duration] <reason>", "Bans user(s) from the server.", permissions.PermissionField["BAN"], cmdBan},
	"bg":       {1, "Usage: /bg <background>", "Sets the area's background.", permissions.PermissionField["CM"], cmdBg},
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
		} else if len(args) < cmd.Args {
			client.SendServerMessage("Not enough arguments.\n" + cmd.Usage)
			return
		}
		cmd.Func(client, args, cmd.Usage)
	} else {
		client.SendServerMessage("You do not have permission to use that command.")
		return
	}
}

// Handles /login
func cmdLogin(client *Client, args []string, usage string) {
	if client.Authenticated() {
		client.SendServerMessage("You are already logged in.")
		return
	}
	auth, perms := db.AuthenticateUser(args[0], []byte(args[1]))
	addToBuffer(client, "AUTH", fmt.Sprintf("Attempted login as %v.", args[0]), true)
	if auth {
		client.SetAuthenticated(true)
		client.SetPerms(perms)
		client.SetModName(args[0])
		client.SendServerMessage("Logged in as moderator.")
		client.SendPacket("AUTH", "1")
		client.SendServerMessage(fmt.Sprintf("Welcome, %v.", args[0]))
		addToBuffer(client, "AUTH", fmt.Sprintf("Logged in as %v.", args[0]), true)
		return
	}
	client.SendPacket("AUTH", "0")
	addToBuffer(client, "AUTH", fmt.Sprintf("Failed login as %v.", args[0]), true)
}

// Handles /logout
func cmdLogout(client *Client, _ []string, _ string) {
	if !client.Authenticated() {
		client.SendServerMessage("Invalid command.")
	}
	client.RemoveAuth()
}

// Handles /mkusr
func cmdMakeUser(client *Client, args []string, usage string) {
	if db.UserExists(args[0]) {
		client.SendServerMessage("User already exists.")
		return
	}

	role, err := getRole(args[2])
	if err != nil {
		client.SendServerMessage("Invalid role.")
		return
	}
	err = db.CreateUser(args[0], []byte(args[1]), role.GetPermissions())
	if err != nil {
		logger.LogError(err.Error())
		client.SendServerMessage("Invalid username/password.")
		return
	}
	client.SendServerMessage("User created.")
}

// Handles /rmusr
func cmdRemoveUser(client *Client, args []string, usage string) {
	if !db.UserExists(args[0]) {
		client.SendServerMessage("User does not exist.")
		return
	}
	err := db.RemoveUser(args[0])
	if err != nil {
		client.SendServerMessage("Failed to remove user.")
		logger.LogError(err.Error())
		return
	}
	client.SendServerMessage("Removed user.")

	for c := range clients.GetAllClients() {
		if c.Authenticated() && c.ModName() == args[0] {
			c.RemoveAuth()
		}
	}
}

// Handles /setrole
func cmdChangeRole(client *Client, args []string, usage string) {
	role, err := getRole(args[1])
	if err != nil {
		client.SendServerMessage("Invalid role.")
		return
	}

	if !db.UserExists(args[0]) {
		client.SendServerMessage("User does not exist.")
		return
	}

	err = db.ChangePermissions(args[0], role.GetPermissions())
	if err != nil {
		client.SendServerMessage("Failed to change permissions.")
		logger.LogError(err.Error())
		return
	}
	client.SendServerMessage("Role updated.")

	for c := range clients.GetAllClients() {
		if c.Authenticated() && c.ModName() == args[0] {
			c.SetPerms(role.GetPermissions())
		}
	}
}

// Handles /kick
func cmdKick(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	uids := &[]string{}
	ipids := &[]string{}
	flags.Var(&cmdParamList{uids}, "u", "")
	flags.Var(&cmdParamList{ipids}, "i", "")
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	var usedList *[]string
	var useUid bool
	if len(*uids) > 0 {
		useUid = true
		usedList = uids
	} else if len(*ipids) > 0 {
		usedList = ipids
	} else {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	toKick := getKBList(usedList, useUid)
	var count int
	var report string
	reason := strings.Join(flags.Args(), " ")
	for _, c := range toKick {
		report += c.Ipid() + ", "
		c.SendPacket("KK", reason)
		c.conn.Close()
		count++
	}
	report = strings.TrimSuffix(report, ", ")
	addToBuffer(client, "CMD", fmt.Sprintf("Kicked %v from server for reason: %v.", report, reason), true)
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
	sendPlayerArup()
}

// Handles /ban
func cmdBan(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	uids := &[]string{}
	ipids := &[]string{}
	flags.Var(&cmdParamList{uids}, "u", "")
	flags.Var(&cmdParamList{ipids}, "i", "")
	duration := flags.String("d", config.BanLen, "")
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	var useUid bool
	var usedList *[]string
	if len(*uids) > 0 {
		useUid = true
		usedList = uids
	} else if len(*ipids) > 0 {
		usedList = ipids
	} else {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	banTime, reason := time.Now().UTC().Unix(), strings.Join(flags.Args(), " ")
	var until int64
	if strings.ToLower(*duration) == "perma" {
		until = -1
	} else {
		parsedDur, err := str2duration.ParseDuration(*duration)
		if err != nil {
			client.SendServerMessage("Failed to ban: Cannot parse duration.")
			return
		}
		until = time.Now().UTC().Add(parsedDur).Unix()
	}

	toBan := getKBList(usedList, useUid)
	var count int
	var report string
	for _, c := range toBan {
		id, err := db.AddBan(c.Ipid(), c.Hdid(), banTime, until, reason, client.ModName())
		if err != nil {
			continue
		}
		var untilS string
		if until == -1 {
			untilS = "∞"
		} else {
			untilS = time.Unix(until, 0).UTC().Format("02 Jan 2006 15:04 MST")
		}
		if !strings.Contains(report, c.Ipid()) {
			report += c.Ipid() + ", "
		}
		client.SendPacket("KB", fmt.Sprintf("%v\nUntil: %v\nID: %v", reason, untilS, id))
		c.conn.Close()
		count++
	}
	report = strings.TrimSuffix(report, ", ")
	addToBuffer(client, "CMD", fmt.Sprintf("Kicked and banned %v from server for %v: %v.", report, *duration, reason), true)
	client.SendServerMessage(fmt.Sprintf("Kicked and banned %v clients.", count))
	sendPlayerArup()
}

// Handles /kickarea
func cmdAreaKick(client *Client, args []string, usage string) {
	if client.Area() == areas[0] {
		client.SendServerMessage("Failed to kick: Cannot kick a user from area 0.")
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
		client.ChangeArea(areas[0])
		c.SendServerMessage("You were kicked from the area!")
		count++
	}
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
}

func cmdBg(client *Client, args []string, usage string) {
	if !sliceutil.ContainsString(backgrounds, args[0]) {
		client.SendServerMessage("Invalid background.")
		return
	}
	client.Area().SetBackground(args[0])
	writeToArea(client.Area(), "BN", args[0])
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the background to %v.", client.OOCName(), args[0]))
}

// Handles /about
func cmdAbout(client *Client, _ []string, _ string) {
	client.SendServerMessage(fmt.Sprintf("Running Athena version %v.\nAthena is open source software; for documentation, bug reports, and source code, see: %v",
		version, "https://github.com/MangosArentLiterature/Athena."))
}
