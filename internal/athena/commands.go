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

type Command struct {
	handler  func(client *Client, args []string, usage string)
	minArgs  int
	usage    string
	desc     string
	reqPerms uint64
}

var Commands map[string]Command

func initCommands() {
	Commands = map[string]Command{
		"about": {
			handler:  cmdAbout,
			minArgs:  0,
			usage:    "Usage: /about",
			desc:     "Prints Athena version information.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"allowcms": {
			handler:  cmdAllowCMs,
			minArgs:  1,
			usage:    "Usage: /allowcms <true|false>",
			desc:     "Toggles allowing CMs on or off.",
			reqPerms: permissions.PermissionField["MODIFY_AREA"],
		},
		"allowiniswap": {
			handler:  cmdAllowIniswap,
			minArgs:  1,
			usage:    "Usage: /allowiniswap <true|false>",
			desc:     "Toggles iniswapping on or off.",
			reqPerms: permissions.PermissionField["MODIFY_AREA"],
		},
		"areainfo": {
			handler:  cmdAreaInfo,
			minArgs:  0,
			usage:    "Usage: /areainfo",
			desc:     "Prints area settings.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"ban": {
			handler:  cmdBan,
			minArgs:  3,
			usage:    "Usage: /ban -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... [-d duration] <reason>",
			desc:     "Bans user(s) from the server.",
			reqPerms: permissions.PermissionField["BAN"],
		},
		"bg": {
			handler:  cmdBg,
			minArgs:  1,
			usage:    "Usage: /bg <background>",
			desc:     "Sets the area's background.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"charselect": {
			handler:  cmdCharSelect,
			minArgs:  0,
			usage:    "Usage: /charselect [uid1],[uid2]...",
			desc:     "Return to character select.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"cm": {
			handler:  cmdCM,
			minArgs:  0,
			usage:    "Usage: /cm [uid1],[uid2]...",
			desc:     "Promote to area CM.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"doc": {
			handler:  cmdDoc,
			minArgs:  0,
			usage:    "Usage: /doc [-c] [doc]\n-c: Clear the doc.",
			desc:     "Prints or sets the area's document.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"editban": {
			handler:  cmdEditBan,
			minArgs:  2,
			usage:    "Usage: /editban [-d duration] [-r reason] <id1>,<id2>...",
			desc:     "Changes the reason of ban(s).",
			reqPerms: permissions.PermissionField["BAN"],
		},
		"evimode": {
			handler:  cmdSetEviMod,
			minArgs:  1,
			usage:    "Usage: /evimode <mode>",
			desc:     "Sets the area's evidence mode.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"forcebglist": {
			handler:  cmdForceBGList,
			minArgs:  1,
			usage:    "Usage: /forcebglist <true|false>",
			desc:     "Toggles enforcing the server BG list on or off.",
			reqPerms: permissions.PermissionField["MODIFY_AREA"],
		},
		"getban": {
			handler:  cmdGetBan,
			minArgs:  0,
			usage:    "Usage: /getban [-b banid | -i ipid]",
			desc:     "Prints ban(s) matching the search parameters, or prints the 5 most recent bans.",
			reqPerms: permissions.PermissionField["BAN_INFO"],
		},
		"global": {
			handler:  cmdGlobal,
			minArgs:  1,
			usage:    "Usage: /global <message>",
			desc:     "Sends a global message.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"invite": {
			handler:  cmdInvite,
			minArgs:  1,
			usage:    "Usage: /invite <uid1>,<uid2>...",
			desc:     "Invites user(s) to the current area.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"kick": {
			handler:  cmdKick,
			minArgs:  3,
			usage:    "Usage: /kick -u <uid1>,<uid2>... | -i <ipid1>,<ipid2>... <reason>",
			desc:     "Kicks user(s) from the server.",
			reqPerms: permissions.PermissionField["KICK"],
		},
		"kickarea": {
			handler:  cmdAreaKick,
			minArgs:  1,
			usage:    "Usage: /kickarea <uid1>,<uid2>...",
			desc:     "Kicks user(s) from the current area.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"lock": {
			handler:  cmdLock,
			minArgs:  0,
			usage:    "Usage: /lock [-s]\n-s: Sets the area to be spectatable.",
			desc:     "Locks the current area or sets it to spectatable.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"lockbg": {
			handler:  cmdLockBG,
			minArgs:  1,
			usage:    "Usage: /lockbg <true|false>",
			desc:     "Toggles locking the BG on or off.",
			reqPerms: permissions.PermissionField["MODIFY_AREA"],
		},
		"lockmusic": {
			handler:  cmdLockMusic,
			minArgs:  1,
			usage:    "Usage: /lockmusic <true|false>",
			desc:     "Toggles CM only music on or off.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"log": {
			handler:  cmdLog,
			minArgs:  1,
			usage:    "Usage: /log <area>",
			desc:     "Prints an area's log buffer.",
			reqPerms: permissions.PermissionField["LOG"],
		},
		"login": {
			handler:  cmdLogin,
			minArgs:  2,
			usage:    "Usage: /login <username> <password>",
			desc:     "Logs in as moderator.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"logout": {
			handler:  cmdLogout,
			minArgs:  0,
			usage:    "Usage: /logout",
			desc:     "Logs out as moderator.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"mkusr": {
			handler:  cmdMakeUser,
			minArgs:  3,
			usage:    "Usage: /mkusr <username> <password> <role>",
			desc:     "Creates a new moderator user.",
			reqPerms: permissions.PermissionField["ADMIN"],
		},
		"mod": {
			handler:  cmdMod,
			minArgs:  1,
			usage:    "Usage: /mod [-g] <message>\n-g: Send the message globally.",
			desc:     "Sends a message speaking officially as a moderator.",
			reqPerms: permissions.PermissionField["MOD_SPEAK"],
		},
		"modchat": {
			handler:  cmdModChat,
			minArgs:  1,
			usage:    "Usage: /modchat <message>",
			desc:     "Sends a message to other moderators.",
			reqPerms: permissions.PermissionField["MOD_CHAT"],
		},
		"motd": {
			handler:  cmdMotd,
			minArgs:  0,
			usage:    "Usage /motd",
			desc:     "Sends the server's message of the day.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"move": {
			handler:  cmdMove,
			minArgs:  1,
			usage:    "Usage: /move [-u <uid1,<uid2>...] <area>",
			desc:     "Moves to an area.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"mute": {
			handler:  cmdMute,
			minArgs:  1,
			usage:    "Usage: /mute [-ic][-ooc][-m][-j][-d duration][-r reason] <uid1>,<uid2>...\n-ic: Mute IC.\n-ooc: Mute OOC.\n-m: Mute music.\n-j: Mute judge.",
			desc:     "Mutes users(s) from IC, OOC, changing music, and/or judge controls.",
			reqPerms: permissions.PermissionField["MUTE"],
		},
		"narrator": {
			handler:  cmdNarrator,
			minArgs:  0,
			usage:    "Usage: /narrator",
			desc:     "Toggles narrator mode on or off.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"nointpres": {
			handler:  cmdNoIntPres,
			minArgs:  1,
			usage:    "Usage: /nointpres <true|false>",
			desc:     "Toggles non-interrupting preanims in the current area on or off.",
			reqPerms: permissions.PermissionField["MODIFY_AREA"],
		},
		"parrot": {
			handler:  cmdParrot,
			minArgs:  1,
			usage:    "Usage: /parrot [-d duration][-r reason] <uid1>,<uid2>...",
			desc:     "Parrots user(s).",
			reqPerms: permissions.PermissionField["MUTE"],
		},
		"play": {
			handler:  cmdPlay,
			minArgs:  1,
			usage:    "Usage: /play <song>",
			desc:     "Plays a song.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"players": {
			handler:  cmdPlayers,
			minArgs:  0,
			usage:    "Usage: /players [-a]\n-a: Target all areas.",
			desc:     "Shows players in the current or all areas.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"pm": {
			handler:  cmdPM,
			minArgs:  2,
			usage:    "Usage: /pm <uid1>,<uid2>... <message>",
			desc:     "Sends a private message.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"rmusr": {
			handler:  cmdRemoveUser,
			minArgs:  1,
			usage:    "Usage: /rmusr <username>",
			desc:     "Removes a moderator user.",
			reqPerms: permissions.PermissionField["ADMIN"],
		},
		"roll": {
			handler:  cmdRoll,
			minArgs:  1,
			usage:    "Usage: /roll [-p] <dice>d<sides>\n-p: Sets the roll to be private.",
			desc:     "Rolls dice.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"setrole": {
			handler:  cmdChangeRole,
			minArgs:  2,
			usage:    "Usage: /setrole <username> <role>",
			desc:     "Changes a moderator user's role.",
			reqPerms: permissions.PermissionField["ADMIN"],
		},
		"status": {
			handler:  cmdStatus,
			minArgs:  1,
			usage:    "Usage: /status <status>",
			desc:     "Sets the current area's status.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"swapevi": {
			handler:  cmdSwapEvi,
			minArgs:  2,
			usage:    "Usage: /swapevi <id1> <id2>",
			desc:     "Swaps index of evidence.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"testimony": {
			handler:  cmdTestimony,
			minArgs:  0,
			usage:    "Usage /testimony <record|stop|play|update|insert|delete>",
			desc:     "Updates the current area's testimony recorder, or prints current testimony.",
			reqPerms: permissions.PermissionField["NONE"],
		},
		"unban": {
			handler:  cmdUnban,
			minArgs:  1,
			usage:    "Usage: /unban <id1>,<id2>...",
			desc:     "Nullifies ban(s).",
			reqPerms: permissions.PermissionField["BAN"],
		},
		"uncm": {
			handler:  cmdUnCM,
			minArgs:  0,
			usage:    "Usage: /uncm [uid1],[uid2]...",
			desc:     "Removes CM(s) from the current area.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"uninvite": {
			handler:  cmdUninvite,
			minArgs:  1,
			usage:    "Usage: /uninvite <uid1>,<uid2>...",
			desc:     "Uninvites user(s) from the current area.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"unlock": {
			handler:  cmdUnlock,
			minArgs:  0,
			usage:    "Usage: /unlock",
			desc:     "Unlocks the current area.",
			reqPerms: permissions.PermissionField["CM"],
		},
		"unmute": {
			handler:  cmdUnmute,
			minArgs:  1,
			usage:    "Usage: /unmute <uid1>,<uid2>...",
			desc:     "Unmutes user(s).",
			reqPerms: permissions.PermissionField["MUTE"],
		},
	}
}

// ParseCommand calls the appropriate function for a given command.
func ParseCommand(client *Client, command string, args []string) {
	if command == "help" {
		var s []string
		for name, cmd := range Commands {
			if permissions.HasPermission(client.Perms(), cmd.reqPerms) || (cmd.reqPerms == permissions.PermissionField["CM"] && client.Area().HasCM(client.Uid())) {
				s = append(s, fmt.Sprintf("- /%v: %v", name, cmd.desc))
			}
		}
		sort.Strings(s)
		client.SendServerMessage("Recognized commands:\n" + strings.Join(s, "\n") + "\n\nTo view detailed usage on a command, do /<command> -h")
		return
	}

	cmd := Commands[command]
	if cmd.handler == nil {
		client.SendServerMessage("Invalid command.")
		return
	} else if permissions.HasPermission(client.Perms(), cmd.reqPerms) || (cmd.reqPerms == permissions.PermissionField["CM"] && client.Area().HasCM(client.Uid())) {
		if sliceutil.ContainsString(args, "-h") {
			client.SendServerMessage(cmd.usage)
			return
		} else if len(args) < cmd.minArgs {
			client.SendServerMessage("Not enough arguments.\n" + cmd.usage)
			return
		}
		cmd.handler(client, args, cmd.usage)
	} else {
		client.SendServerMessage("You do not have permission to use that command.")
		return
	}
}

// Handles /about
func cmdAbout(client *Client, _ []string, _ string) {
	client.SendServerMessage(fmt.Sprintf("Running Athena version %v.\nAthena is open source software; for documentation, bug reports, and source code, see: %v",
		version, "https://github.com/MangosArentLiterature/Athena."))
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

// Handles /areainfo
func cmdAreaInfo(client *Client, _ []string, _ string) {
	out := fmt.Sprintf("\nBG: %v\nEvi mode: %v\nAllow iniswap: %v\nNon-interrupting pres: %v\nCMs allowed: %v\nForce BG list: %v\nBG locked: %v\nMusic locked: %v",
		client.Area().Background(), client.Area().EvidenceMode().String(), client.Area().IniswapAllowed(), client.Area().NoInterrupt(),
		client.Area().CMsAllowed(), client.Area().ForceBGList(), client.Area().LockBG(), client.Area().LockMusic())
	client.SendServerMessage(out)
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

// Handles /bg
func cmdBg(client *Client, args []string, _ string) {
	if client.Area().LockBG() && !permissions.HasPermission(client.Perms(), permissions.PermissionField["MODIFY_AREA"]) {
		client.SendServerMessage("You do not have permission to change the background in this area.")
		return
	}

	arg := strings.Join(args, " ")

	if client.Area().ForceBGList() && !sliceutil.ContainsString(backgrounds, arg) {
		client.SendServerMessage("Invalid background.")
		return
	}
	client.Area().SetBackground(arg)
	writeToArea(client.Area(), "BN", arg)
	sendAreaServerMessage(client.Area(), fmt.Sprintf("%v set the background to %v.", client.OOCName(), arg))
	addToBuffer(client, "CMD", fmt.Sprintf("Set BG to %v.", arg), false)
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

// Handles /editban
func cmdEditBan(client *Client, args []string, usage string) {
	flags := flag.NewFlagSet("", 0)
	flags.SetOutput(io.Discard)
	duration := flags.String("d", "", "")
	reason := flags.String("r", "", "")
	flags.Parse(args)
	useDur := *duration != ""
	useReason := *reason != ""

	if len(flags.Args()) == 0 || (!useDur && !useReason) {
		client.SendServerMessage("Not enough arguments:\n" + usage)
		return
	}
	toUpdate := strings.Split(flags.Arg(0), ",")
	var until int64
	if useDur {
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
	}

	var report string
	for _, s := range toUpdate {
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		if useDur {
			err = db.UpdateDuration(id, until)
			if err != nil {
				continue
			}
		}
		if useReason {
			err = db.UpdateReason(id, *reason)
			if err != nil {
				continue
			}
		}
		report += fmt.Sprintf("%v, ", s)
	}
	report = strings.TrimSuffix(report, ", ")
	client.SendServerMessage(fmt.Sprintf("Updated bans: %v", report))
	if useDur {
		addToBuffer(client, "CMD", fmt.Sprintf("Edited bans: %v to duration: %v.", report, duration), true)
	}
	if useReason {
		addToBuffer(client, "CMD", fmt.Sprintf("Edited bans: %v to reason: %v.", report, reason), true)
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

// Handles /global
func cmdGlobal(client *Client, args []string, _ string) {
	if !client.CanSpeakOOC() {
		client.SendServerMessage("You are muted from sending OOC messages.")
		return
	}
	writeToAll("CT", fmt.Sprintf("[GLOBAL] %v", client.OOCName()), strings.Join(args, " "), "1")
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

// Handles /modchat
func cmdModChat(client *Client, args []string, _ string) {
	msg := strings.Join(args, " ")
	for c := range clients.GetAllClients() {
		if permissions.HasPermission(c.Perms(), permissions.PermissionField["MOD_CHAT"]) {
			c.SendPacket("CT", fmt.Sprintf("[MODCHAT] %v", client.OOCName()), msg, "1")
		}
	}
}

// Handles /motd
func cmdMotd(client *Client, _ []string, _ string) {
	client.SendServerMessage(config.Motd)
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

// Handles /narrator
func cmdNarrator(client *Client, _ []string, _ string) {
	client.ToggleNarrator()
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

// Handles /parrot
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

// Handles /pm
func cmdPM(client *Client, args []string, _ string) {
	msg := strings.Join(args[1:], " ")
	toPM := getUidList(strings.Split(args[0], ","))
	for _, c := range toPM {
		c.SendPacket("CT", fmt.Sprintf("[PM] %v", client.OOCName()), msg, "1")
	}
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
