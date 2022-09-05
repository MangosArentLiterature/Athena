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
	"io"
	"math/rand"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MangosArentLiterature/Athena/internal/area"
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
	//admin commands
	"mkusr":   {3, "Usage: /mkusr <username> <password> <role>", "Creates a new moderator user.", permissions.PermissionField["ADMIN"], cmdMakeUser},
	"rmusr":   {1, "Usage: /rmusr <username>", "Removes a moderator user.", permissions.PermissionField["ADMIN"], cmdRemoveUser},
	"setrole": {2, "Usage: /setrole <username> <role>", "Changes a moderator user's role.", permissions.PermissionField["ADMIN"], cmdChangeRole},

	//general commands
	"about":   {0, "Usage: /about", "Prints Athena version information.", permissions.PermissionField["NONE"], cmdAbout},
	"move":    {1, "Usage: /move [-u <uid1,<uid2>...] <area>\n-u: Uid(s).", "Moves to an area.", permissions.PermissionField["NONE"], cmdMove},
	"pm":      {2, "Usage: /pm <uid1>,<uid2>... <message>", "Sends a private message.", permissions.PermissionField["NONE"], cmdPM},
	"global":  {1, "Usage: /global <message>", "Sends a global message.", permissions.PermissionField["NONE"], cmdGlobal},
	"roll":    {1, "Usage: /roll [-p] <dice>d<sides>\n-p: Private", "Rolls dice.", permissions.PermissionField["NONE"], cmdRoll},
	"motd":    {0, "Usage /motd", "Sends the server's message of the day.", permissions.PermissionField["NONE"], cmdMotd},
	"players": {0, "Usage: /players [-a]\n-a: All.", "Shows players in the current or all areas.", permissions.PermissionField["NONE"], cmdPlayers},

	//area commands
	"bg":           {1, "Usage: /bg <background>", "Sets background.", permissions.PermissionField["CM"], cmdBg},
	"status":       {1, "Usage: /status <status>", "Sets status.", permissions.PermissionField["CM"], cmdStatus},
	"cm":           {0, "Usage: /cm [uid1],[uid2]...", "Adds CM(s).", permissions.PermissionField["NONE"], cmdCM},
	"uncm":         {0, "Usage: /uncm [uid1],[uid2]...", "Removes CM(s).", permissions.PermissionField["CM"], cmdUnCM},
	"lock":         {0, "Usage: /lock [-s]\n-s: Spectatable.", "Locks the area or sets it to spectatable.", permissions.PermissionField["CM"], cmdLock},
	"unlock":       {0, "Usage: /unlock", "Unlocks the area.", permissions.PermissionField["CM"], cmdUnlock},
	"invite":       {1, "Usage: /invite <uid1>,<uid2>...", "Invites user(s).", permissions.PermissionField["CM"], cmdInvite},
	"uninvite":     {1, "Usage: /uninvite <uid1>,<uid2>...", "Uninvites user(s).", permissions.PermissionField["CM"], cmdUninvite},
	"evimode":      {1, "Usage: /evimode <mode>", "Sets the evidence mode.", permissions.PermissionField["CM"], cmdSetEviMod},
	"kickarea":     {1, "Usage: /kickarea <uid1>,<uid2>...", "Kicks user(s) from the area.", permissions.PermissionField["CM"], cmdAreaKick},
	"swapevi":      {2, "Usage: /swapevi <id1> <id2>", "Swaps index of evidence.", permissions.PermissionField["NONE"], cmdSwapEvi},
	"nointpres":    {1, "Usage: /nointpres <true|false>", "Toggles non-interrupting preanims.", permissions.PermissionField["MODIFY_AREA"], cmdNoIntPres},
	"allowiniswap": {1, "Usage: /allowiniswap <true|false>", "Toggles iniswapping.", permissions.PermissionField["MODIFY_AREA"], cmdAllowIniswap},
	"forcebglist":  {1, "Usage: /forcebglist <true|false>", "Toggles enforcing the server BG list.", permissions.PermissionField["MODIFY_AREA"], cmdForceBGList},
	"allowcms":     {1, "Usage: /allowcms <true|false>", "Toggles allowing CMs.", permissions.PermissionField["MODIFY_AREA"], cmdAllowCMs},
	"lockbg":       {1, "Usage: /lockbg <true|false>", "Toggles locking the BG.", permissions.PermissionField["MODIFY_AREA"], cmdLockBG},
	"lockmusic":    {1, "Usage: /lockmusic <true|false>", "Toggles making music CM only.", permissions.PermissionField["CM"], cmdLockMusic},
	"charselect":   {0, "Usage: /charselect [uid1],[uid2]...", "Moves back to character select.", permissions.PermissionField["NONE"], cmdCharSelect},
	"areainfo":     {0, "Usage: /areainfo", "Shows area information.", permissions.PermissionField["NONE"], cmdAreaInfo},
	"doc":          {0, "Usage: /doc [-c] [doc]\n-c: Clear.", "Gets or sets the doc.", permissions.PermissionField["NONE"], cmdDoc},
	"play":         {1, "Usage: /play <song>", "Plays a song.", permissions.PermissionField["CM"], cmdPlay},
	"testimony":    {0, "Usage /testimony <record|stop|play|update|insert|delete>", "Modifies or prints recorded testimony.", permissions.PermissionField["NONE"], cmdTestimony},

	//mod commands
	"login":   {2, "Usage: /login <username> <password>", "Logs in as moderator.", permissions.PermissionField["NONE"], cmdLogin},
	"logout":  {0, "Usage: /logout", "Logs out as moderator.", permissions.PermissionField["NONE"], cmdLogout},
	"kick":    {3, "Usage: /kick -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... <reason>\n-u: Uid(s).\n-i: Ipid(s).", "Kicks user(s) from the server.", permissions.PermissionField["KICK"], cmdKick},
	"ban":     {3, "Usage: /ban -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... [-d duration] <reason>\n-u: Uid(s).\n-i: Ipid(s).\n-d: Duration", "Bans user(s) from the server.", permissions.PermissionField["BAN"], cmdBan},
	"mod":     {1, "Usage: /mod [-g] <message>\n-g: Global.", "Sends a message speaking officially as a moderator.", permissions.PermissionField["MOD_SPEAK"], cmdMod},
	"getban":  {0, "Usage: /getban [-b banid | -i ipid]\n-b: BanID.\n-i: IPID.", "Searches bans or gets the most recent bans.", permissions.PermissionField["BAN_INFO"], cmdGetBan},
	"unban":   {1, "Usage: /unban <id1>,<id2>...", "Nullifies a ban.", permissions.PermissionField["BAN"], cmdUnban},
	"editban": {2, "Usage: /editban <id1>,<id2>... <reason>", "Changes the reason of ban(s).", permissions.PermissionField["BAN"], cmdEditBan},
	"modchat": {1, "Usage: /modchat <message>", "Sends a message to the mod chat.", permissions.PermissionField["MOD_CHAT"], cmdModChat},
	"mute":    {1, "Usage: /mute [-ic][-ooc][-m][-j][-d duration][-r reason] <uid1>,<uid2>...\n-ic: IC.\n-ooc: OOC.\n-m: Music.\n-j: Judge.\n-d: Duration.\n -r: Reason.", "Mutes users(s) from IC/OOC/Music/Judge.", permissions.PermissionField["MUTE"], cmdMute},
	"unmute":  {1, "Usage: /unmute <uid1>,<uid2>...", "Unmutes user(s).", permissions.PermissionField["MUTE"], cmdUnmute},
	"parrot":  {1, "Usage: /parrot [-d duration][-r reason] <uid1>,<uid2>...\n-d: Duration.\n-r: Reason.", "Parrots user(s).", permissions.PermissionField["MUTE"], cmdParrot},
	"log":     {1, "Usage: /log <area>", "Gets an area's log buffer.", permissions.PermissionField["LOG"], cmdLog},
}

// ParseCommand calls the appropriate function for a given command.
func ParseCommand(client *Client, command string, args []string) {
	if command == "help" {
		var s []string
		for name, attr := range commands {
			if permissions.HasPermission(client.Perms(), attr.Permission) || (attr.Permission == permissions.PermissionField["CM"] && client.Area().HasCM(client.Uid())) {
				s = append(s, fmt.Sprintf("- /%v: %v", name, attr.Desc))
			}
		}
		sort.Strings(s)
		client.SendServerMessage("Recognized commands:\n" + strings.Join(s, "\n") + "\n\nTo view detailed usage on a command, do /<command> -h")
		return
	}

	cmd := commands[command]
	if cmd.Func == nil {
		client.SendServerMessage("Invalid command.")
		return
	} else if permissions.HasPermission(client.Perms(), cmd.Permission) || (cmd.Permission == permissions.PermissionField["CM"] && client.Area().HasCM(client.Uid())) {
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
func cmdLogin(client *Client, args []string, _ string) {
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
		client.SendServerMessage("You are not logged in.")
	}
	addToBuffer(client, "AUTH", fmt.Sprintf("Logged out as %v.", client.ModName()), true)
	client.RemoveAuth()
}

// Handles /mkusr
func cmdMakeUser(client *Client, args []string, _ string) {
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
	addToBuffer(client, "CMD", fmt.Sprintf("Created user %v.", args[0]), true)
}

// Handles /rmusr
func cmdRemoveUser(client *Client, args []string, _ string) {
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
	addToBuffer(client, "CMD", fmt.Sprintf("Removed user %v.", args[0]), true)
}

// Handles /setrole
func cmdChangeRole(client *Client, args []string, _ string) {
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
	addToBuffer(client, "CMD", fmt.Sprintf("Updated role of %v to %v.", args[0], args[1]), true)
}

// Handles /kick
func cmdKick(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	uids := &[]string{}
	ipids := &[]string{}
	flags.Var(&cmdParamList{uids}, "u", "")
	flags.Var(&cmdParamList{ipids}, "i", "")
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

	var toKick []*Client
	if len(*uids) > 0 {
		toKick = getUidList(*uids)
	} else if len(*ipids) > 0 {
		toKick = getIpidList(*ipids)
	} else {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}

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
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
	sendPlayerArup()
	addToBuffer(client, "CMD", fmt.Sprintf("Kicked %v from server for reason: %v.", report, reason), true)
}

// Handles /ban
func cmdBan(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
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

	var toBan []*Client
	if len(*uids) > 0 {
		toBan = getUidList(*uids)
	} else if len(*ipids) > 0 {
		toBan = getIpidList(*ipids)
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
		c.SendPacket("KB", fmt.Sprintf("%v\nUntil: %v\nID: %v", reason, untilS, id))
		c.conn.Close()
		count++
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Banned %v clients.", count))
	sendPlayerArup()
	addToBuffer(client, "CMD", fmt.Sprintf("Banned %v from server for %v: %v.", report, *duration, reason), true)
}

// Handles /kickarea
func cmdAreaKick(client *Client, args []string, _ string) {
	if client.Area() == areas[0] {
		client.SendServerMessage("Failed to kick: Cannot kick a user from area 0.")
		return
	}
	toKick := getUidList(strings.Split(args[0], ","))

	var count int
	var report string
	for _, c := range toKick {
		if c.Area() != client.Area() || permissions.HasPermission(c.Perms(), permissions.PermissionField["BYPASS_LOCK"]) {
			continue
		}
		if c == client {
			client.SendServerMessage("You can't kick yourself from the area.")
			continue
		}
		c.ChangeArea(areas[0])
		c.SendServerMessage("You were kicked from the area!")
		count++
		report += fmt.Sprintf("%v, ", c.Uid())
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Kicked %v clients.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Kicked %v from area.", report), false)
}

// Handles /bg
func cmdBg(client *Client, args []string, _ string) {
	if client.Area().LockBG() && !permissions.HasPermission(client.Perms(), permissions.PermissionField["MODIFY_AREA"]) {
		client.SendServerMessage("You do not have permission to change the background in this area.")
		return
	}

	if client.Area().ForceBGList() && !sliceutil.ContainsString(backgrounds, args[0]) {
		client.SendServerMessage("Invalid background.")
		return
	}
	client.Area().SetBackground(args[0])
	writeToArea(client.Area(), "BN", args[0])
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the background to %v.", client.OOCName(), args[0]))
	addToBuffer(client, "CMD", fmt.Sprintf("Set BG to %v.", args[0]), false)
}

// Handles /about
func cmdAbout(client *Client, _ []string, _ string) {
	client.SendServerMessage(fmt.Sprintf("Running Athena version %v.\nAthena is open source software; for documentation, bug reports, and source code, see: %v",
		version, "https://github.com/MangosArentLiterature/Athena."))
}

// Handles /cm
func cmdCM(client *Client, args []string, _ string) {
	if client.CharID() == -1 {
		client.SendServerMessage("You are spectating; you cannot become a CM.")
		return
	} else if !client.Area().CMsAllowed() && !client.HasCMPermission() {
		client.SendServerMessage("You do not have permission to use that command.")
		return
	}

	if len(args) == 0 {
		if client.Area().HasCM(client.Uid()) {
			client.SendServerMessage("You are already a CM in this area.")
			return
		} else if len(client.Area().CMs()) > 0 && !permissions.HasPermission(client.Perms(), permissions.PermissionField["CM"]) {
			client.SendServerMessage("This area already has a CM.")
			return
		}
		client.Area().AddCM(client.Uid())
		client.SendServerMessage("Successfully became a CM.")
		addToBuffer(client, "CMD", "CMed self.", false)
	} else {
		if !client.HasCMPermission() {
			client.SendServerMessage("You do not have permission to use that command.")
			return
		}
		toCM := getUidList(strings.Split(args[0], ","))
		var count int
		var report string
		for _, c := range toCM {
			if c.Area() != client.Area() || c.Area().HasCM(c.Uid()) {
				continue
			}
			c.Area().AddCM(c.Uid())
			c.SendServerMessage("You have become a CM in this area.")
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
		report = strings.TrimSuffix(report, ", ")
		client.SendServerMessage(fmt.Sprintf("CMed %v users.", count))
		addToBuffer(client, "CMD", fmt.Sprintf("CMed %v.", report), false)
	}
	sendCMArup()
}

// Handles /uncm
func cmdUnCM(client *Client, args []string, _ string) {
	if len(args) == 0 {
		if !client.Area().HasCM(client.Uid()) {
			client.SendServerMessage("You are not a CM in this area.")
			return
		}
		client.Area().RemoveCM(client.Uid())
		client.SendServerMessage("You are no longer a CM in this area.")
		addToBuffer(client, "CMD", "Un-CMed self.", false)
	} else {
		toCM := getUidList(strings.Split(args[0], ","))
		var count int
		var report string
		for _, c := range toCM {
			if c.Area() != client.Area() || !c.Area().HasCM(c.Uid()) {
				continue
			}
			c.Area().RemoveCM(c.Uid())
			c.SendServerMessage("You are no longer a CM in this area.")
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
		report = strings.TrimSuffix(report, ", ")
		client.SendServerMessage(fmt.Sprintf("Un-CMed %v users.", count))
		addToBuffer(client, "CMD", fmt.Sprintf("Un-CMed %v.", report), false)
	}
	sendCMArup()
}

// Handles /status
func cmdStatus(client *Client, args []string, _ string) {
	switch strings.ToLower(args[0]) {
	case "idle":
		client.Area().SetStatus(area.StatusIdle)
	case "looking-for-players":
		client.Area().SetStatus(area.StatusPlayers)
	case "casing":
		client.Area().SetStatus(area.StatusCasing)
	case "recess":
		client.Area().SetStatus(area.StatusRecess)
	case "rp":
		client.Area().SetStatus(area.StatusRP)
	case "gaming":
		client.Area().SetStatus(area.StatusGaming)
	default:
		client.SendServerMessage("Status not recognized. Recognized statuses: idle, looking-for-players, casing, recess, rp, gaming")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the status to %v.", client.OOCName(), args[0]))
	sendStatusArup()
	addToBuffer(client, "CMD", fmt.Sprintf("Set the status to %v.", args[0]), false)
}

// Handles /lock
func cmdLock(client *Client, args []string, _ string) {
	if sliceutil.ContainsString(args, "-s") { // Set area to spectatable.
		client.Area().SetLock(area.LockSpectatable)
		sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the area to spectatable.", client.OOCName()))
		addToBuffer(client, "CMD", "Set the area to spectatable.", false)
	} else { // Normal lock.
		if client.Area().Lock() == area.LockLocked {
			client.SendServerMessage("This area is already locked.")
			return
		} else if client.Area() == areas[0] {
			client.SendServerMessage("You cannot lock area 0.")
			return
		}
		client.Area().SetLock(area.LockLocked)
		sendAreaServerMessage(client.Area(), fmt.Sprintf("%v locked the area.", client.OOCName()))
		addToBuffer(client, "CMD", "Locked the area.", false)
	}
	for c := range clients.GetAllClients() {
		if c.Area() == client.Area() {
			c.Area().AddInvited(c.Uid())
		}
	}
	sendLockArup()
}

// Handles /unlock
func cmdUnlock(client *Client, _ []string, _ string) {
	if client.Area().Lock() == area.LockFree {
		client.SendServerMessage("This area is not locked.")
		return
	}
	client.Area().SetLock(area.LockFree)
	client.Area().ClearInvited()
	sendLockArup()
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v unlocked the area.", client.OOCName()))
	addToBuffer(client, "CMD", "Unlocked the area.", false)
}

// Handles /invite
func cmdInvite(client *Client, args []string, _ string) {
	if client.Area().Lock() == area.LockFree {
		client.SendServerMessage("This area is unlocked.")
		return
	}
	toInvite := getUidList(strings.Split(args[0], ","))
	var count int
	var report string
	for _, c := range toInvite {
		if client.Area().AddInvited(c.Uid()) {
			c.SendServerMessage(fmt.Sprintf("You were invited to area %v.", client.Area().Name()))
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Invited %v users.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Invited %v to the area.", report), false)
}

// Handles /uninvite
func cmdUninvite(client *Client, args []string, _ string) {
	if client.Area().Lock() == area.LockFree {
		client.SendServerMessage("This area is unlocked.")
		return
	}
	toUninvite := getUidList(strings.Split(args[0], ","))
	var count int
	var report string
	for _, c := range toUninvite {
		if c == client || client.Area().HasCM(c.Uid()) {
			continue
		}
		if client.Area().RemoveInvited(c.Uid()) {
			if c.Area() == client.Area() && client.Area().Lock() == area.LockLocked && !permissions.HasPermission(c.Perms(), permissions.PermissionField["BYPASS_LOCK"]) {
				c.SendServerMessage("You were kicked from the area!")
				c.ChangeArea(areas[0])
			}
			c.SendServerMessage(fmt.Sprintf("You were uninvited from area %v.", client.Area().Name()))
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Uninvited %v users.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Uninvited %v to the area.", report), false)
}

// Handles swapevi
func cmdSwapEvi(client *Client, args []string, _ string) {
	if !client.CanAlterEvidence() {
		client.SendServerMessage("You are not allowed to alter evidence in this area.")
		return
	}
	evi1, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	evi2, err := strconv.Atoi(args[1])
	if err != nil {
		return
	}
	if client.Area().SwapEvidence(evi1, evi2) {
		client.SendServerMessage("Evidence swapped.")
		writeToArea(client.Area(), "LE", client.Area().Evidence()...)
		addToBuffer(client, "CMD", fmt.Sprintf("Swapped posistions of evidence %v and %v.", evi1, evi2), false)
	} else {
		client.SendServerMessage("Invalid arguments.")
	}
}

// Handles /evimode
func cmdSetEviMod(client *Client, args []string, _ string) {
	if !client.CanAlterEvidence() {
		client.SendServerMessage("You are not allowed to change the evidence mode.")
		return
	}
	switch args[0] {
	case "mods":
		if !permissions.HasPermission(client.Perms(), permissions.PermissionField["MOD_EVI"]) {
			client.SendServerMessage("You do not have permission for this evidence mode.")
			return
		}
		client.Area().SetEvidenceMode(area.EviMods)
	case "cms":
		client.Area().SetEvidenceMode(area.EviCMs)
	case "any":
		client.Area().SetEvidenceMode(area.EviAny)
	default:
		client.SendServerMessage("Invalid evidence mode.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the evidence mode to %v.", client.OOCName(), args[0]))
	addToBuffer(client, "CMD", fmt.Sprintf("Set the evidence mode to %v.", args[0]), false)
}

// Handles /nointpres
func cmdNoIntPres(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetNoInterrupt(true)
		result = "enabled"
	case "false":
		client.Area().SetNoInterrupt(false)
		result = "disabled"
	default:
		client.SendServerMessage("Argument not recognized.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v non-interrupting preanims in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set non-interrupting preanims to %v.", args[0]), false)
}

// Handles /allowiniswap
func cmdAllowIniswap(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetIniswapAllowed(true)
		result = "enabled"
	case "false":
		client.Area().SetIniswapAllowed(false)
		result = "disabled"
	default:
		client.SendServerMessage("Argument not recognized.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v iniswapping in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set iniswapping to %v.", args[0]), false)
}

// Handles /forcebglist
func cmdForceBGList(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetForceBGList(true)
		result = "enforced"
	case "false":
		client.Area().SetForceBGList(false)
		result = "unenforced"
	default:
		client.SendServerMessage("Argument not recognized.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v the BG list in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set the BG list to %v.", args[0]), false)
}

// Handles /lockbg
func cmdLockBG(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetLockBG(true)
		result = "locked"
	case "false":
		client.Area().SetLockBG(false)
		result = "unlocked"
	default:
		client.SendServerMessage("Argument not recognized.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v the background in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set the background to %v.", args[0]), false)
}

// Handles /lockmusic
func cmdLockMusic(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetLockMusic(true)
		result = "enabled"
	case "false":
		client.Area().SetLockMusic(false)
		result = "disabled"
	default:
		client.SendServerMessage("Argument not recognized.")
		return
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v CM-only music in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set CM-only music list to %v.", args[0]), false)
}

// Handles /allowcms
func cmdAllowCMs(client *Client, args []string, _ string) {
	var result string
	switch args[0] {
	case "true":
		client.Area().SetCMsAllowed(true)
		result = "allowed"
	case "false":
		client.Area().SetCMsAllowed(false)
		result = "disallowed"
	default:
		client.SendServerMessage("Argument not recognized.")
	}
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v has %v CMs in this area.", client.OOCName(), result))
	addToBuffer(client, "CMD", fmt.Sprintf("Set allowing CMs to %v.", args[0]), false)
}

// Handles /move
func cmdMove(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	uids := &[]string{}
	flags.Var(&cmdParamList{uids}, "u", "")
	flags.Parse(args)

	if len(flags.Args()) < 1 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	areaID, err := strconv.Atoi(flags.Arg(0))
	if err != nil || areaID < 0 || areaID > len(areas)-1 {
		client.SendServerMessage("Invalid area.")
		return
	}
	wantedArea := areas[areaID]

	if len(*uids) > 0 {
		if !permissions.HasPermission(client.Perms(), permissions.PermissionField["MOVE_USERS"]) {
			client.SendServerMessage("You do not have permission to use that command.")
			return
		}
		toMove := getUidList(*uids)
		var count int
		var report string
		for _, c := range toMove {
			if !c.ChangeArea(wantedArea) {
				continue
			}
			c.SendServerMessage(fmt.Sprintf("You were moved to %v.", wantedArea.Name()))
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
		report = strings.TrimSuffix(report, ", ")
		client.SendServerMessage(fmt.Sprintf("Moved %v users.", count))
		addToBuffer(client, "CMD", fmt.Sprintf("Moved %v to %v.", report, wantedArea.Name()), false)
	} else {
		if !client.ChangeArea(wantedArea) {
			client.SendServerMessage("You are not invited to that area.")
		}
		client.SendServerMessage(fmt.Sprintf("Moved to %v.", wantedArea.Name()))
	}
}

// Handles /charselect
func cmdCharSelect(client *Client, args []string, _ string) {
	if len(args) == 0 {
		client.ChangeCharacter(-1)
		client.SendPacket("DONE")
	} else {
		if !client.HasCMPermission() {
			client.SendServerMessage("You do not have permission to use that command.")
			return
		}
		toChange := getUidList(strings.Split(args[0], ","))
		var count int
		var report string
		for _, c := range toChange {
			if c.Area() != client.Area() || c.CharID() == -1 {
				continue
			}
			c.ChangeCharacter(-1)
			c.SendPacket("DONE")
			c.SendServerMessage("You were moved back to character select.")
			count++
			report += fmt.Sprintf("%v, ", c.Uid())
		}
		report = strings.TrimSuffix(report, ", ")
		client.SendServerMessage(fmt.Sprintf("Moved %v users to character select.", count))
		addToBuffer(client, "CMD", fmt.Sprintf("Moved %v to character select.", report), false)
	}
}

// Handles /players
func cmdPlayers(client *Client, args []string, _ string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	all := flags.Bool("a", false, "")
	flags.Parse(args)
	out := "\nPlayers\n----------\n"
	entry := func(c *Client, auth bool) string {
		s := fmt.Sprintf("[%v] %v\n", c.Uid(), c.CurrentCharacter())
		if auth {
			if c.Authenticated() {
				s += fmt.Sprintf("Mod: %v\n", c.ModName())
			}
			s += fmt.Sprintf("IPID: %v\n", c.Ipid())
		}
		if c.OOCName() != "" {
			s += fmt.Sprintf("OOC: %v\n", c.OOCName())
		}
		return s
	}
	if *all {
		for _, a := range areas {
			out += fmt.Sprintf("%v:\n%v players online.\n", a.Name(), a.PlayerCount())
			for c := range clients.GetAllClients() {
				if c.Area() == a {
					out += entry(c, client.Authenticated())
				}
			}
			out += "----------\n"
		}
	} else {
		out += fmt.Sprintf("%v:\n%v players online.\n", client.Area().Name(), client.Area().PlayerCount())
		for c := range clients.GetAllClients() {
			if c.Area() == client.Area() {
				out += entry(c, client.Authenticated())
			}
		}
	}
	client.SendServerMessage(out)
}

// Handles /areainfo
func cmdAreaInfo(client *Client, _ []string, _ string) {
	out := fmt.Sprintf("\nBG: %v\nEvi mode: %v\nAllow iniswap: %v\nNon-interrupting pres: %v\nCMs allowed: %v\nForce BG list: %v\nBG locked: %v\nMusic locked: %v",
		client.Area().Background(), client.Area().EvidenceMode().String(), client.Area().IniswapAllowed(), client.Area().NoInterrupt(),
		client.Area().CMsAllowed(), client.Area().ForceBGList(), client.Area().LockBG(), client.Area().LockMusic())
	client.SendServerMessage(out)
}

// Handles /pm
func cmdPM(client *Client, args []string, _ string) {
	msg := strings.Join(args[1:], " ")
	toPM := getUidList(strings.Split(args[0], ","))
	for _, c := range toPM {
		c.SendPacket("CT", fmt.Sprintf("[PM] %v", client.OOCName()), msg, "1")
	}
}

// Handles /global
func cmdGlobal(client *Client, args []string, _ string) {
	if !client.CanSpeakOOC() {
		client.SendServerMessage("You are muted from sending OOC messages.")
		return
	}
	writeToAll("CT", fmt.Sprintf("[GLOBAL] %v", client.OOCName()), strings.Join(args, " "), "1")
}

// Handles /roll
func cmdRoll(client *Client, args []string, _ string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	private := flags.Bool("p", false, "")
	flags.Parse(args)
	b, _ := regexp.MatchString("([[:digit:]])d([[:digit:]])", flags.Arg(0))
	if !b {
		client.SendServerMessage("Argument not recognized.")
		return
	}
	s := strings.Split(flags.Arg(0), "d")
	num, _ := strconv.Atoi(s[0])
	sides, _ := strconv.Atoi(s[1])
	if num <= 0 || num > config.MaxDice || sides <= 0 || sides > config.MaxSide {
		client.SendServerMessage("Invalid num/side.")
		return
	}
	var result []string
	gen := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < num; i++ {
		result = append(result, fmt.Sprint(gen.Intn(sides)+1))
	}
	if *private {
		client.SendServerMessage(fmt.Sprintf("Results: %v.", strings.Join(result, ", ")))
	} else {
		sendAreaServerMessage(client.Area(), fmt.Sprintf("%v rolled %v. Results: %v.", client.OOCName(), flags.Arg(0), strings.Join(result, ", ")))
	}
	addToBuffer(client, "CMD", fmt.Sprintf("Rolled %v.", flags.Arg(0)), false)
}

// Handles /motd
func cmdMotd(client *Client, _ []string, _ string) {
	client.SendServerMessage(config.Motd)
}

// Handles /mod
func cmdMod(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	global := flags.Bool("g", false, "")
	flags.Parse(args)
	if len(flags.Args()) == 0 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	msg := strings.Join(flags.Args(), " ")
	if *global {
		writeToAll("CT", fmt.Sprintf("[MOD] [GLOBAL] %v", client.OOCName()), msg, "1")
	} else {
		writeToArea(client.Area(), "CT", fmt.Sprintf("[MOD] %v", client.OOCName()), msg, "1")
	}
	addToBuffer(client, "OOC", msg, false)
}

// Handles /getban
func cmdGetBan(client *Client, args []string, _ string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	banid := flags.Int("b", -1, "")
	ipid := flags.String("i", "", "")
	flags.Parse(args)
	s := "Bans:\n----------"
	entry := func(b db.BanInfo) string {
		var d string
		if b.Duration == -1 {
			d = "∞"
		} else {
			d = time.Unix(b.Duration, 0).UTC().Format("02 Jan 2006 15:04 MST")
		}

		return fmt.Sprintf("\nID: %v\nIPID: %v\nHDID: %v\nBanned on: %v\nUntil: %v\nReason: %v\nModerator: %v\n----------",
			b.Id, b.Ipid, b.Hdid, time.Unix(b.Time, 0).UTC().Format("02 Jan 2006 15:04 MST"), d, b.Reason, b.Moderator)
	}
	if *banid > 0 {
		b, err := db.GetBan(db.BANID, *banid)
		if err != nil || len(b) == 0 {
			client.SendServerMessage("No ban with that ID exists.")
			return
		}
		s += entry(b[0])
	} else if *ipid != "" {
		bans, err := db.GetBan(db.IPID, *ipid)
		if err != nil || len(bans) == 0 {
			client.SendServerMessage("No bans with that IPID exist.")
			return
		}
		for _, b := range bans {
			s += entry(b)
		}
	} else {
		bans, err := db.GetRecentBans()
		if err != nil {
			logger.LogErrorf("while getting recent bans: %v", err)
			client.SendServerMessage("An unexpected error occured.")
			return
		}
		for _, b := range bans {
			s += entry(b)
		}
	}
	client.SendServerMessage(s)
}

// Handles /unban
func cmdUnban(client *Client, args []string, _ string) {
	toUnban := strings.Split(args[0], ",")
	var report string
	for _, s := range toUnban {
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		err = db.UnBan(id)
		if err != nil {
			continue
		}
		report += fmt.Sprintf("%v, ", s)
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Nullified bans: %v", report))
	addToBuffer(client, "CMD", fmt.Sprintf("Nullified bans: %v", report), true)
}

// Handles /editban
func cmdEditBan(client *Client, args []string, _ string) {
	toUpdate := strings.Split(args[0], ",")
	reason := strings.Join(args[1:], " ")
	var report string
	for _, s := range toUpdate {
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		err = db.UpdateBan(id, reason)
		if err != nil {
			continue
		}
		report += fmt.Sprintf("%v, ", s)
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Updated bans: %v", report))
	addToBuffer(client, "CMD", fmt.Sprintf("Nullified bans: %v to reason: %v.", report, reason), true)
}

// Handles /modchat
func cmdModChat(client *Client, args []string, _ string) {
	msg := strings.Join(args, " ")
	for c := range clients.GetAllClients() {
		if permissions.HasPermission(c.Perms(), permissions.PermissionField["MOD_CHAT"]) {
			c.SendPacket("CT", fmt.Sprintf("[MODCHAT] %v", client.OOCName()), msg, "1")
		}
	}
}

// Handles /mute
func cmdMute(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	reason := flags.String("r", "", "")
	music := flags.Bool("m", false, "")
	jud := flags.Bool("j", false, "")
	ic := flags.Bool("ic", false, "")
	ooc := flags.Bool("ooc", false, "")
	duration := flags.Int("d", -1, "")
	flags.Parse(args)

	var m MuteState
	switch {
	case *ic && *ooc:
		m = ICOOCMuted
	case *ic:
		m = ICMuted
	case *ooc:
		m = OOCMuted
	case *music:
		m = MusicMuted
	case *jud:
		m = JudMuted
	default:
		m = ICMuted
	}
	msg := fmt.Sprintf("You have been muted from %v", m.String())
	if *duration != -1 {
		msg += fmt.Sprintf(" for %v seconds", *duration)
	}
	if *reason != "" {
		msg += " for reason: " + *reason
	}
	if len(flags.Args()) == 0 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	toMute := getUidList(strings.Split(flags.Arg(0), ","))
	var count int
	var report string
	for _, c := range toMute {
		if c.Muted() == m {
			continue
		}
		c.SetMuted(m)
		if *duration == -1 {
			c.SetUnmuteTime(time.Time{})
		} else {
			c.SetUnmuteTime(time.Now().UTC().Add(time.Duration(*duration) * time.Second))
		}
		c.SendServerMessage(msg)
		count++
		report += fmt.Sprintf("%v, ", c.Uid())
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Muted %v clients.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Muted %v.", report), false)
}

// Handles /unmute
func cmdUnmute(client *Client, args []string, _ string) {
	toUnmute := getUidList(strings.Split(args[0], ","))
	var count int
	var report string
	for _, c := range toUnmute {
		if c.Muted() == Unmuted {
			continue
		}
		c.SetMuted(Unmuted)
		c.SendServerMessage("You have been unmuted.")
		count++
		report += fmt.Sprintf("%v, ", c.Uid())
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Unmuted %v clients.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Unmuted %v.", report), false)
}

// Handles /Parrot
func cmdParrot(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	reason := flags.String("r", "", "")
	duration := flags.Int("d", -1, "")
	flags.Parse(args)
	msg := "You have been turned into a parrot"
	if *duration != -1 {
		msg += fmt.Sprintf(" for %v seconds", *duration)
	}
	if *reason != "" {
		msg += " for reason: " + *reason
	}
	if len(flags.Args()) == 0 {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	toParrot := getUidList(strings.Split(flags.Arg(0), ","))
	var count int
	var report string
	for _, c := range toParrot {
		if c.Muted() != Unmuted {
			continue
		}
		c.SetMuted(ParrotMuted)
		if *duration == -1 {
			c.SetUnmuteTime(time.Time{})
		} else {
			c.SetUnmuteTime(time.Now().UTC().Add(time.Duration(*duration) * time.Second))
		}
		c.SendServerMessage(msg)
		count++
		report += fmt.Sprintf("%v, ", c.Uid())
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Parroted %v clients.", count))
	addToBuffer(client, "CMD", fmt.Sprintf("Parroted %v.", report), false)
}

// Handles /log
func cmdLog(client *Client, args []string, _ string) {
	wantedArea, err := strconv.Atoi(args[0])
	if err != nil {
		client.SendServerMessage("Invalid area.")
		return
	}
	for i, a := range areas {
		if i == wantedArea {
			client.SendServerMessage(strings.Join(a.Buffer(), "\n"))
			return
		}
	}
	client.SendServerMessage("Invalid area.")
}

// Handles /doc
func cmdDoc(client *Client, args []string, _ string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	clear := flags.Bool("c", false, "")
	flags.Parse(args)
	if len(args) == 0 {
		if client.Area().Doc() == "" {
			client.SendServerMessage("This area does not have a doc set.")
			return
		}
		client.SendServerMessage(client.Area().Doc())
		return
	} else {
		if !client.HasCMPermission() {
			client.SendServerMessage("You do not have permission to change the doc.")
			return
		} else if *clear {
			client.Area().SetDoc("")
			sendAreaServerMessage(client.Area(), fmt.Sprintf("%v cleared the doc.", client.OOCName()))
			return
		} else if len(flags.Args()) != 0 {
			client.Area().SetDoc(flags.Arg(0))
			sendAreaServerMessage(client.Area(), fmt.Sprintf("%v updated the doc.", client.OOCName()))
			return
		}
	}
}

// Handles /play
func cmdPlay(client *Client, args []string, _ string) {
	if !client.CanChangeMusic() {
		client.SendServerMessage("You are not allowed to change the music in this area.")
		return
	}
	s := strings.Join(args, " ")

	// Check if the song we got is a URL for streaming
	if _, err := url.ParseRequestURI(s); err == nil {
		s, err = url.QueryUnescape(s) // Unescape any URL encoding
		if err != nil {
			client.SendServerMessage("Error parsing URL.")
			return
		}
	}
	writeToArea(client.Area(), "MC", s, fmt.Sprint(client.CharID()), client.Showname(), "1", "0")
}

// Handles /testimony
func cmdTestimony(client *Client, args []string, _ string) {
	if len(args) == 0 {
		if !client.Area().HasTestimony() {
			client.SendServerMessage("This area has no recorded testimony.")
			return
		}
		client.SendServerMessage(strings.Join(client.area.Testimony(), "\n"))
		return
	} else if !client.HasCMPermission() {
		client.SendServerMessage("You do not have permission to use that command.")
		return
	}
	switch args[0] {
	case "record":
		if client.Area().TstState() != area.TRIdle {
			client.SendServerMessage("The recorder is currently active.")
			return
		}
		client.Area().TstClear()
		client.Area().SetTstState(area.TRRecording)
		client.SendServerMessage("Recording testimony.")
	case "stop":
		client.Area().SetTstState(area.TRIdle)
		client.SendServerMessage("Recorder stopped.")
		client.Area().TstJump(0)
		writeToArea(client.Area(), "RT", "testimony1#1")
	case "play":
		if !client.Area().HasTestimony() {
			client.SendServerMessage("No testimony recorded.")
			return
		}
		client.Area().SetTstState(area.TRPlayback)
		client.SendServerMessage("Playing testimony.")
		writeToArea(client.Area(), "RT", "testimony2")
		writeToArea(client.Area(), "MS", client.Area().CurrentTstStatement())
	case "update":
		if client.Area().TstState() != area.TRPlayback {
			client.SendServerMessage("The recorder is not active.")
			return
		}
		client.Area().SetTstState(area.TRUpdating)
	case "insert":
		if client.Area().TstState() != area.TRPlayback {
			client.SendServerMessage("The recorder is not active.")
			return
		}
		client.Area().SetTstState(area.TRInserting)
	case "delete":
		if client.Area().TstState() != area.TRPlayback {
			client.SendServerMessage("The recorder is not active.")
			return
		}
		if client.Area().CurrentTstIndex() > 0 {
			err := client.Area().TstRemove()
			if err != nil {
				client.SendServerMessage("Failed to delete statement.")
			}
		}
	}
}
