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
	"strconv"
	"strings"

	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/packet"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
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
	"DE":      {0, true, pktRemoveEvi},
	"EE":      {4, true, pktEditEvi},
	"CH":      {0, false, pktPing},
}

// Handles HI#%
func pktHdid(client *Client, p *packet.Packet) {
	if strings.TrimSpace(p.Body[0]) == "" || client.uid != -1 {
		return
	}

	// Athena does not store the client's raw HDID, but rather, it's MD5 hash.
	// This is done not only for privacy reasons, but to ensure stored HDIDs will be a reasonable length.
	hash := md5.Sum([]byte(p.Body[0]))
	client.hdid = base64.StdEncoding.EncodeToString(hash[:])
	client.write(fmt.Sprintf("ID#0#Athena#%v#%%", version)) // Why does the client need this? Nobody knows.
}

// Handles ID#%
func pktId(client *Client, p *packet.Packet) {
	if client.uid != -1 {
		return
	}
	client.version = p.Body[1]
	client.write(fmt.Sprintf("PN#%v#%v#%v#%%", players.GetPlayerCount(), config.MaxPlayers, config.Desc))
	// god this is cursed
	client.write("FL#noencryption#yellowtext#prezoom#flipping#customobjections#fastloading#deskmod#evidence#cccc_ic_support#arup#casing_alerts#looping_sfx#additive#effects#y_offset#expanded_desk_mods#auth_packet#%")
}

// Handles askchaa#%
func pktResCount(client *Client, _ *packet.Packet) {
	if client.uid != -1 {
		return
	}
	if players.GetPlayerCount() >= config.MaxPlayers {
		logger.LogInfo("Player limit reached")
		client.write("BD#This server is full#%")
		client.conn.Close()
		return
	}
	client.write(fmt.Sprintf("SI#%v#%v#%v#%%", len(characters), 0, len(music)))
}

// Handles RC#%
func pktReqChar(client *Client, _ *packet.Packet) {
	client.write(fmt.Sprintf("SC#%v#%%", strings.Join(characters, "#")))
}

// Handles RM#%
func pktReqAM(client *Client, _ *packet.Packet) {
	client.write(fmt.Sprintf("SM#%v#%v#%%", areaNames, strings.Join(music, "#")))
}

// Handles RD#%
func pktReqDone(client *Client, _ *packet.Packet) {
	if client.uid != -1 {
		return
	}
	client.uid = uids.GetUid()
	players.AddPlayer()
	client.area = areas[0]
	client.area.AddChar(-1)
	sendPlayerArup()
	def, pro := client.area.GetHP()
	client.write(fmt.Sprintf("LE#%v#%%", strings.Join(client.area.GetEvidence(), "#")))
	client.write(fmt.Sprintf("CharsCheck#%v#%%", strings.Join(client.area.GetTaken(), "#")))
	client.write(fmt.Sprintf("HP#1#%v#%%", def))
	client.write(fmt.Sprintf("HP#2#%v#%%", pro))
	logger.LogInfof("Client (IPID:%v UID:%v) joined the server", client.ipid, client.uid)
	client.write("DONE#%")
}

// Handles CC#%
func pktChangeChar(client *Client, p *packet.Packet) {
	if client.uid == -1 {
		return
	}
	newid, err := strconv.Atoi(p.Body[1])
	if err != nil {
		return
	}
	if client.area.SwitchChar(client.char, newid) {
		client.char = newid
		client.write(fmt.Sprintf("PV#0#CID#%v#%%", newid))
		writeToArea(fmt.Sprintf("CharsCheck#%v#%%", strings.Join(client.area.GetTaken(), "#")), client.area)
	}
}

// Handles MS#%
func pktIC(client *Client, p *packet.Packet) {
	p.Body[4] = strings.TrimSpace(p.Body[4])
	if client.char == -1 {
		return
	} else if len(p.Body[4]) > config.MaxMsg {
		client.sendServerMessage("Your message exceeds the maximum message length!")
		return
	} else if p.Body[4] == client.lastmsg {
		return
	}
	args := len(p.Body)
	newPacket, _ := packet.NewPacket("MS")

	// Validate desk_mod
	if !sliceutil.Contains([]string{"chat", "0", "1", "2", "3", "4", "5"}, p.Body[0]) {
		return
	}

	//emote_modifier
	if p.Body[7] == "4" {
		p.Body[7] = "6"
	}
	if !sliceutil.Contains([]string{"0", "1", "2", "5", "6"}, p.Body[7]) {
		return
	}

	// Validate char_id
	if p.Body[8] != strconv.Itoa(client.char) {
		return
	}

	newPacket.Body = p.Body[:15] // Append all base args

	if args >= 18 { //2.6+ args
		extargs := []string{p.Body[15], p.Body[16], "", "", p.Body[17], p.Body[18]}
		newPacket.Body = append(newPacket.Body, extargs...)

		if args == 26 {
			//2.8+ args
			newPacket.Body = append(newPacket.Body, p.Body[19:]...)
		}
	}
	client.lastmsg = p.Body[4]
	writeToArea(newPacket.String(), client.area)
}

// Handles MC#%
func pktAM(client *Client, p *packet.Packet) {
	if client.uid == -1 || strconv.Itoa(client.char) != p.Body[1] {
		return
	}

	if sliceutil.Contains(music, p.Body[0]) && client.char != -1 {
		song := p.Body[0]
		effects := "0"
		if !strings.ContainsRune(p.Body[0], '.') {
			song = "~stop.mp3"
		}
		if len(p.Body) >= 4 {
			effects = p.Body[3]
		}
		writeToArea(fmt.Sprintf("MC#%v#%v#%v#1#0#%v#%%", song, p.Body[1], p.Body[2], effects), client.area)
	} else if strings.Contains(areaNames, p.Body[0]) {
		if p.Body[0] == client.area.Name {
			return
		}
		for _, area := range areas {
			if area.Name == p.Body[0] && area.AddChar(client.char) {
				client.area.RemoveChar(client.char)
				client.area = area
				def, pro := client.area.GetHP()
				client.write(fmt.Sprintf("LE#%v#%%", strings.Join(client.area.GetEvidence(), "#")))
				client.write(fmt.Sprintf("HP#1#%v#%%", def))
				client.write(fmt.Sprintf("HP#2#%v#%%", pro))
				sendPlayerArup()
				writeToArea(fmt.Sprintf("CharsCheck#%v#%%", strings.Join(client.area.GetTaken(), "#")), client.area)
			}
		}
	}
}

// Handles HP#%
func pktHP(client *Client, p *packet.Packet) {
	bar, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	value, err := strconv.Atoi(p.Body[1])

	if err != nil {
		return
	}
	if !client.area.SetHP(bar, value) {
		return
	}
	writeToArea(fmt.Sprintf("HP#%v#%%", p.Body[0]), client.area)
}

// Handles RT#%
func pktWTCE(client *Client, p *packet.Packet) {
	if client.uid == -1 {
		return
	}
	writeToArea(fmt.Sprintf("RT#%v#%%", p.Body[0]), client.area)
}

// Handles CT#%
func pktOOC(client *Client, p *packet.Packet) {
	if strings.TrimSpace(p.Body[0]) == "" || p.Body[0] == config.Name {
		client.sendServerMessage("Invalid username.")
		return
	} else if len(p.Body[1]) > config.MaxMsg {
		client.sendServerMessage("Your message exceeds the maximum message length!")
		return
	}
	for c := range clients.GetClients() {
		if c.oocName == p.Body[0] && c != client {
			client.sendServerMessage("That username is already taken.")
			return
		}
	}
	client.oocName = p.Body[0]
	writeToArea(fmt.Sprintf("CT#%v#%v#0#%%", p.Body[0], p.Body[1]), client.area)
}

// Handles PE#%
func pktAddEvi(client *Client, p *packet.Packet) {
	client.area.AddEvidence(strings.Join(p.Body, "&"))
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.area.GetEvidence(), "#")), client.area)
}

// Handles DE#%
func pktRemoveEvi(client *Client, p *packet.Packet) {
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.area.RemoveEvidence(id)
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.area.GetEvidence(), "#")), client.area)
}

// Handles EE#%
func pktEditEvi(client *Client, p *packet.Packet) {
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.area.EditEvidence(id, strings.Join(p.Body[1:], "&"))
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.area.GetEvidence(), "#")), client.area)
}

// Handles CH#%
func pktPing(client *Client, _ *packet.Packet) {
	client.write("CHECK#%")
}
