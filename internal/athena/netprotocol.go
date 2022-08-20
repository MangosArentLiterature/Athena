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
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/packet"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
	"github.com/MangosArentLiterature/Athena/internal/webhook"
)

// Documentation for AO2's network protocol can be found here:
// https://github.com/AttorneyOnline/docs/blob/master/docs/development/network.md

type pktMapValue struct {
	Args     int
	MustJoin bool
	Func     func(client *Client, p *packet.Packet)
}

var PacketMap = map[string]pktMapValue{
	"HI":      {1, false, pktHdid},
	"ID":      {2, false, pktId},
	"askchaa": {0, false, pktResCount},
	"RC":      {0, false, pktReqChar},
	"RM":      {0, false, pktReqAM},
	"RD":      {0, false, pktReqDone},
	"CC":      {3, true, pktChangeChar},
	"MS":      {15, true, pktIC},
	"MC":      {2, true, pktAM},
	"HP":      {2, true, pktHP},
	"RT":      {1, true, pktWTCE},
	"CT":      {2, true, pktOOC},
	"PE":      {3, true, pktAddEvi},
	"DE":      {1, true, pktRemoveEvi},
	"EE":      {4, true, pktEditEvi},
	"CH":      {0, false, pktPing},
	"ZZ":      {0, true, pktModcall},
	"SETCASE": {7, true, pktSetCase},
	"CASEA":   {6, true, pktCaseAnn},
}

// Handles HI#%
func pktHdid(client *Client, p *packet.Packet) {
	if strings.TrimSpace(p.Body[0]) == "" || client.Uid() != -1 || client.Hdid() != "" {
		return
	}

	// Athena does not store the client's raw HDID, but rather, it's MD5 hash.
	// This is done not only for privacy reasons, but to ensure stored HDIDs will be a reasonable length.
	hash := md5.Sum([]byte(decode(p.Body[0])))
	client.SetHdid(base64.StdEncoding.EncodeToString(hash[:]))
	client.SetHdid(client.Hdid()[:len(client.Hdid())-2]) // Removes the trailing padding.

	client.CheckBanned(db.HDID)

	client.SendPacket("ID", "0", "Athena", encode(version)) // Why does the client need this? Nobody knows.
}

// Handles ID#%
func pktId(client *Client, p *packet.Packet) {
	if client.Uid() != -1 {
		return
	}
	client.SendPacket("PN", strconv.Itoa(players.GetPlayerCount()), strconv.Itoa(config.MaxPlayers), encode(config.Desc))
	client.SendPacket("FL", "noencryption", "yellowtext", "prezoom", "flipping", "customobjections",
		"fastloading", "deskmod", "evidence", "cccc_ic_support", "arup", "casing_alerts",
		"modcall_reason", "looping_sfx", "additive", "effects", "y_offset", "expanded_desk_mods", "auth_packet") // god this is cursed

	if config.AssetURL != "" {
		client.SendPacket("ASS", config.AssetURL)
	}
}

// Handles askchaa#%
func pktResCount(client *Client, _ *packet.Packet) {
	if client.Uid() != -1 || client.Hdid() == "" {
		return
	}
	if players.GetPlayerCount() >= config.MaxPlayers {
		logger.LogInfo("Player limit reached")
		client.SendPacket("BD", "This server is currently full.")
		client.conn.Close()
		return
	}
	client.joining = true // This simply exists to prevent skipping the askchaa#% packet and bypassing the player count check.
	client.SendPacket("SI", strconv.Itoa(len(characters)), strconv.Itoa(len(areas[0].Evidence())), strconv.Itoa(len(music)))
}

// Handles RC#%
func pktReqChar(client *Client, _ *packet.Packet) {
	client.SendPacket("SC", characters...)
}

// Handles RM#%
func pktReqAM(client *Client, _ *packet.Packet) {
	client.write(fmt.Sprintf("SM#%v#%v#%%", areaNames, strings.Join(music, "#")))
}

// Handles RD#%
func pktReqDone(client *Client, _ *packet.Packet) {
	if client.Uid() != -1 || !client.joining || client.Hdid() == "" {
		return
	}
	client.SetUid(uids.GetUid())
	players.AddPlayer()
	updatePlayers <- players.GetPlayerCount()
	client.JoinArea(areas[0])
	client.SendPacket("DONE")
	sendCMArup()
	sendStatusArup()
	sendLockArup()
	logger.LogInfof("Client (IPID:%v UID:%v) joined the server", client.Ipid(), client.Uid())
}

// Handles CC#%
func pktChangeChar(client *Client, p *packet.Packet) {
	newid, err := strconv.Atoi(p.Body[1])
	if err != nil {
		return
	}
	if client.Area().SwitchChar(client.CharID(), newid) {
		client.SetCharID(newid)
		client.SendPacket("PV", "0", "CID", strconv.Itoa(newid))
		writeToArea(client.Area(), "CharsCheck", client.Area().Taken()...)
	}
}

// Handles MS#%
func pktIC(client *Client, p *packet.Packet) {
	// Welcome to the MS packet validation hell.

	if client.CharID() == -1 || !client.CanSpeak() { // Literally 1984
		client.SendServerMessage("You are not allowed to speak in this area.")
		return
	}
	// Clients can send differing numbers of arguments depending on their version.
	// Rather than individually check arguments, we simply copy the arguments that *do* exist.
	// Nonexisting args will simply be blank.
	args := make([]string, 26)
	copy(args, p.Body)

	// The MS#% packet sent from the server has a different number of args than the clients because of pairing.
	// For some godforsaken reason, AO2 places these new arguments in two different spots in the middle of the packet.
	// So two insertions are required.
	args = append(args[:19], args[17:]...)
	args = append(args[:20], args[18:]...)

	client.SetPos(args[5])
	emote_mod, err := strconv.Atoi(args[7])
	if err != nil {
		return
	} else if emote_mod == 4 { // Value of 4 can crash the client.
		args[7] = "6"
	}
	objection, err := strconv.Atoi(strings.Split(args[10], "&")[0])
	if err != nil {
		return
	}
	evi, err := strconv.Atoi(args[11])
	if err != nil {
		return
	}
	text, err := strconv.Atoi(args[14])
	if err != nil {
		return
	}

	if args[22] == "" {
		args[22] = "0"
	}
	if args[23] == "" {
		args[23] = "0"
	}
	if args[24] == "" {
		args[24] = "0"
	}
	if args[28] == "" || client.CharID() != client.Area().LastSpeaker() {
		args[28] = "0"
	}
	if (client.Area().NoInterrupt() && emote_mod != 0) || args[22] == "1" {
		args[22] = "1"
		if emote_mod == 1 || emote_mod == 2 {
			args[7] = "0"
		} else if emote_mod == 6 {
			args[7] = "5"
		}
	}

	switch {
	case !sliceutil.ContainsString([]string{"chat", "0", "1", "2", "3", "4", "5"}, args[0]): // desk_mod
		return
	case !strings.EqualFold(characters[client.CharID()], args[2]) && !client.Area().IniswapAllowed(): // character name
		client.SendServerMessage("Iniswapping is not allowed in this area.")
		return
	case len(decode(p.Body[4])) > config.MaxMsg: // message
		client.SendServerMessage("Your message exceeds the maximum message length!")
		return
	case p.Body[4] == client.LastMsg():
		return
	case emote_mod < 0 || emote_mod > 6:
		return
	case args[8] != strconv.Itoa(client.CharID()): // char_id
		return
	case objection < 0 || objection > 4: // objection_mod
		return
	case evi < 0 || evi > len(client.Area().Evidence()): // evidence
		return
	case args[12] != "0" && args[12] != "1": // flipping
		return
	case args[13] != "0" && args[13] != "1": // realization
		return
	case text < 0 || text > 6: // text color
		return
	case len(args[14]) > 30: // showname
		client.SendServerMessage("Your showname is too long!")
		return
	case args[22] != "0" && args[22] != "1": // non-interrupting preanim
		return
	case args[23] != "0" && args[23] != "1": // sfx looping
		return
	case args[24] != "0" && args[24] != "1": // screenshake
		return
	case args[28] != "0" && args[28] != "1": // additive
		return
	}

	// Pairing validation
	if args[16] != "" && args[16] != "-1" {
		pid, err := strconv.Atoi(strings.Split(args[16], "^")[0])
		if err != nil {
			return
		}
		if pid < 0 || pid > len(characters) || pid == client.CharID() {
			return
		}
		client.SetPairWantedID(pid)
		pairing := false
		for c := range clients.GetAllClients() {
			if c.CharID() == pid && c.Pos() == client.Pos() && c.PairWantedID() == client.CharID() {
				pairinfo := c.PairInfo()
				args[17] = pairinfo.name
				args[18] = pairinfo.emote
				args[20] = pairinfo.offset
				args[21] = pairinfo.flip
				pairing = true
				break
			}
		}
		if !pairing {
			args[16] = "-1^"
		}
	}
	// Offset validation
	if args[19] != "" {
		offsets := strings.Split(decode(args[19]), "&")
		x_offset, err := strconv.Atoi(offsets[0])
		if err != nil {
			return
		} else if x_offset < -100 || x_offset > 100 {
			return
		}
		if len(offsets) > 1 {
			y_offset, err := strconv.Atoi(offsets[0])
			if err != nil {
				return
			} else if y_offset < -100 || y_offset > 100 {
				return
			}
		}
	}

	client.SetPairInfo(args[2], args[3], args[12], args[19])
	client.SetLastMsg(p.Body[4])
	client.Area().SetLastSpeaker(client.CharID())
	writeToArea(client.Area(), "MS", args...)
	addToBuffer(client, "IC", "\""+p.Body[4]+"\"", false)
}

// Handles MC#%
func pktAM(client *Client, p *packet.Packet) {
	// For reasons beyond mortal understanding, this packet serves two purposes: music changes, and area changes.

	if strconv.Itoa(client.CharID()) != p.Body[1] {
		return
	}

	if sliceutil.ContainsString(music, p.Body[0]) {
		if !client.CanChangeMusic() {
			client.SendServerMessage("You are not allowed to change the music in this area.")
			return
		}
		song := p.Body[0]
		name := characters[client.CharID()]
		effects := "0"
		if !strings.ContainsRune(p.Body[0], '.') { // Chosen song is a category, and should stop the music.
			song = "~stop.mp3"
			addToBuffer(client, "MUSIC", "Stopped the music.", false)
		} else {
			addToBuffer(client, "MUSIC", fmt.Sprintf("Changed music to %v.", song), false)
		}
		if len(p.Body) > 2 {
			name = p.Body[2]
		}
		if len(p.Body) > 3 {
			effects = p.Body[3]
		}
		writeToArea(client.Area(), "MC", song, p.Body[1], name, "1", "0", effects)
	} else if strings.Contains(areaNames, p.Body[0]) {
		if decode(p.Body[0]) == client.Area().Name() {
			return
		}
		for _, a := range areas {
			if a.Name() == decode(p.Body[0]) {
				if a.Lock() == area.LockLocked &&
					!sliceutil.ContainsInt(a.Invited(), client.Uid()) &&
					!permissions.HasPermission(client.Perms(), permissions.PermissionField["BYPASS_LOCK"]) {
					client.SendServerMessage("You are not invited to that area.")
					return
				}
				addToBuffer(client, "AREA", "Left area.", false)
				client.ChangeArea(a)
				addToBuffer(client, "AREA", "Joined area.", false)
				return
			}
		}
	}
}

// Handles HP#%
func pktHP(client *Client, p *packet.Packet) {
	if client.CharID() == -1 || !client.CanSpeak() {
		client.SendServerMessage("You are not allowed to change the penalty bar in this area.")
		return
	}
	bar, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	value, err := strconv.Atoi(p.Body[1])

	if err != nil {
		return
	}
	if !client.Area().SetHP(bar, value) {
		return
	}
	writeToArea(client.Area(), "HP", p.Body[0], p.Body[1])

	var side string
	switch bar {
	case 1:
		side = "Defense"
	case 2:
		side = "Prosecution"
	}
	addToBuffer(client, "JUD", fmt.Sprintf("Set %v HP to %v.", side, value), false)
}

// Handles RT#%
func pktWTCE(client *Client, p *packet.Packet) {
	if client.CharID() == -1 || !client.CanSpeak() {
		client.SendServerMessage("You are not allowed to play WT/CE in this area.")
		return
	}
	writeToArea(client.Area(), "RT", p.Body[0])
	addToBuffer(client, "JUD", "Played WT/CE animation.", false)
}

// Handles CT#%
func pktOOC(client *Client, p *packet.Packet) {
	username := decode(strings.TrimSpace(p.Body[0]))
	if username == "" || username == config.Name || len(username) > 30 {
		client.SendServerMessage("Invalid username.")
		return
	} else if len(p.Body[1]) > config.MaxMsg {
		client.SendServerMessage("Your message exceeds the maximum message length!")
		return
	}
	for c := range clients.GetAllClients() {
		if c.OOCName() == p.Body[0] && c != client {
			client.SendServerMessage("That username is already taken.")
			return
		}
	}
	client.SetOocName(username)

	if strings.HasPrefix(p.Body[1], "/") {
		decoded := decode(p.Body[1])
		regex := regexp.MustCompile("^/[a-z]+")
		command := strings.TrimPrefix(regex.FindString(decoded), "/")
		args := strings.Split(strings.Join(regex.Split(decoded, 1), ""), " ")[1:]
		ParseCommand(client, command, args)
		return
	}

	writeToArea(client.Area(), "CT", encode(client.oocName), p.Body[1], "0")
	addToBuffer(client, "OOC", "\""+p.Body[1]+"\"", false)
}

// Handles PE#%
func pktAddEvi(client *Client, p *packet.Packet) {
	if !client.CanAlterEvidence() {
		client.SendServerMessage("You are not allowed to alter evidence in this area.")
		return
	}
	client.Area().AddEvidence(strings.Join(p.Body, "&"))
	writeToArea(client.Area(), "LE", client.Area().Evidence()...)
	addToBuffer(client, "EVI", fmt.Sprintf("Added evidence: %v | %v", p.Body[0], p.Body[1]), false)
}

// Handles DE#%
func pktRemoveEvi(client *Client, p *packet.Packet) {
	if !client.CanAlterEvidence() {
		client.SendServerMessage("You are not allowed to alter evidence in this area.")
		return
	}
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.Area().RemoveEvidence(id)
	writeToArea(client.Area(), "LE", client.Area().Evidence()...)
	addToBuffer(client, "EVI", fmt.Sprintf("Removed evidence %v.", id), false)
}

// Handles EE#%
func pktEditEvi(client *Client, p *packet.Packet) {
	if !client.CanAlterEvidence() {
		client.SendServerMessage("You are not allowed to alter evidence in this area.")
		return
	}
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.Area().EditEvidence(id, strings.Join(p.Body[1:], "&"))
	writeToArea(client.Area(), "LE", client.Area().Evidence()...)
	addToBuffer(client, "EVI", fmt.Sprintf("Updated evidence %v to %v | %v", id, p.Body[1], p.Body[2]), false)
}

// Handles CH#%
func pktPing(client *Client, _ *packet.Packet) {
	client.SendPacket("CHECK")
}

// Handles ZZ#%
func pktModcall(client *Client, p *packet.Packet) {
	var s string
	if len(p.Body) >= 1 {
		s = p.Body[0]
	}
	addToBuffer(client, "MOD", fmt.Sprintf("Called moderator for reason: %v", s), false)
	for c := range clients.GetAllClients() {
		if c.Authenticated() {
			c.SendPacket("ZZ", fmt.Sprintf("[%v] %v (%v): %v", client.Area().Name(), client.CurrentCharacter(), client.Ipid(), s))
		}
	}
	if enableDiscord {
		err := webhook.PostModcall(client.CurrentCharacter(), client.Area().Name(), s)
		if err != nil {
			logger.LogError(err.Error())
		}
	}
	logger.WriteReport(client.Area().Name(), client.Area().Buffer())
}

// Handles SETCASE#%
func pktSetCase(client *Client, p *packet.Packet) {
	for i, r := range p.Body[2:] {
		if i >= 4 {
			break
		}
		b, err := strconv.ParseBool(r)
		if err != nil {
			return
		}
		client.SetRoleAlert(i, b)
	}
}

// Handles CASEA#%
func pktCaseAnn(client *Client, p *packet.Packet) {
	// Let future generations know I spent far too long trying to make this work.
	// Partially because of my own stupidity, and partially because this is the worst packet in AO2.

	if client.CharID() == -1 || !client.HasCMPermission() {
		client.SendServerMessage("You are not allowed to send case alerts in this area.")
		return
	}
	newPacket := fmt.Sprintf("CASEA#CASE ANNOUNCEMENT: %v in %v needs players for %v#%v#1#%%",
		client.CurrentCharacter(), client.Area().Name(), p.Body[0], strings.Join(p.Body[1:], "#")) // Due to a bug, old client versions require this packet to have an extra arg.

	for c := range clients.GetAllClients() {
		if c == client {
			continue
		}
		for i, r := range p.Body[1:] {
			if i >= 4 {
				break
			}
			b, err := strconv.ParseBool(r)
			if err != nil {
				return
			}
			if b && c.AlertRole(i) {
				c.write(newPacket)
				break
			}
		}
	}
}

// decode returns a given string as a decoded AO2 string.
func decode(s string) string {
	return strings.NewReplacer("<percent>", "%", "<num>", "#", "<dollar>", "$", "<and>", "&").Replace(s)
}

// encode returns a string encoded AO2 string.
func encode(s string) string {
	return strings.NewReplacer("%", "<percent>", "#", "<num>", "$", "<dollar>", "&", "<and>").Replace(s)
}
