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
	"fmt"
	"strconv"
	"strings"

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
	if strings.TrimSpace(p.Body[0]) == "" || client.Uid != -1 {
		return
	}
	client.Hdid = p.Body[0]
	fmt.Fprintf(client.Conn, "ID#0#Athena#%v#%%", version) // Why does the client need this? Nobody knows.
}

// Handles ID#%
func pktId(client *Client, p *packet.Packet) {
	if client.Uid != -1 {
		return
	}
	client.Version = p.Body[1]
	fmt.Fprintf(client.Conn, "PN#%v#%v#%v#%%", getPlayerCount(), config.MaxPlayers, config.Desc)
	// god this is cursed
	fmt.Fprint(client.Conn, "FL#noencryption#yellowtext#prezoom#flipping#customobjections#fastloading#deskmod#evidence#cccc_ic_support#arup#casing_alerts#looping_sfx#additive#effects#y_offset#expanded_desk_mods#auth_packet#%")
}

// Handles askchaa#%
func pktResCount(client *Client, _ *packet.Packet) {
	if getPlayerCount() >= config.MaxPlayers {
		fmt.Fprint(client.Conn, "BD#This server is full#%")
		client.Conn.Close()
		return
	}
	fmt.Fprintf(client.Conn, "SI#%v#%v#%v#%%", len(characters), 0, len(music))
}

// Handles RC#%
func pktReqChar(client *Client, _ *packet.Packet) {
	fmt.Fprintf(client.Conn, "SC#%v#%%", strings.Join(characters, "#"))
}

// Handles RM#%
func pktReqAM(client *Client, _ *packet.Packet) {
	fmt.Fprintf(client.Conn, "SM#%v#%v#%%", areaNames, strings.Join(music, "#"))
}

// Handles RD#%
func pktReqDone(client *Client, _ *packet.Packet) {
	if client.Uid != -1 {
		return
	}
	client.takeUid()
	playerCountMu.Lock()
	playerCount++
	playerCountMu.Unlock()
	client.Area = areas[0]
	client.Area.Join(-1)
	sendPlayerArup()
	def, pro := client.Area.GetHP()
	fmt.Fprintf(client.Conn, "LE#%v#%%", strings.Join(client.Area.GetEvidence(), "#"))
	fmt.Fprintf(client.Conn, "CharsCheck#%v#%%", strings.Join(client.Area.GetTaken(), "#"))
	fmt.Fprintf(client.Conn, "HP#1#%v#%%", def)
	fmt.Fprintf(client.Conn, "HP#2#%v#%%", pro)

	fmt.Fprint(client.Conn, "DONE#%")
}

// Handles CC#%
func pktChangeChar(client *Client, p *packet.Packet) {
	if client.Uid == -1 {
		return
	}
	newid, err := strconv.Atoi(p.Body[1])
	if err != nil {
		return
	}
	if client.Area.Switch(client.Char, newid) {
		client.Char = newid
		fmt.Fprintf(client.Conn, "PV#0#CID#%v#%%", newid)
	}
}

// Handles MS#%
func pktIC(client *Client, p *packet.Packet) {
	if client.Char == -1 {
		client.sendServerMessage("You're a spectator, you can't chat here.")
		return
	} else if len(p.Body[4]) > config.MaxMsg {
		client.sendServerMessage("Your message exceeds the maximum message length!")
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
	if p.Body[8] != strconv.Itoa(client.Char) {
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
	writeToArea(newPacket.String(), client.Area)
}

// Handles MC#%
func pktAM(client *Client, p *packet.Packet) {
	if client.Uid == -1 || strconv.Itoa(client.Char) != p.Body[1] {
		return
	}

	if sliceutil.Contains(music, p.Body[0]) && client.Char != -1 {
		song := p.Body[0]
		effects := "0"
		if !strings.ContainsRune(p.Body[0], '.') {
			song = "~stop.mp3"
		}
		if len(p.Body) >= 4 {
			effects = p.Body[3]
		}
		writeToArea(fmt.Sprintf("MC#%v#%v#%v#1#0#%v#%%", song, p.Body[1], p.Body[2], effects), client.Area)
	} else if strings.Contains(areaNames, p.Body[0]) {
		for _, area := range areas {
			if area.Name == p.Body[0] && area.Join(client.Char) {
				client.Area = area
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
	if !client.Area.SetHP(bar, value) {
		return
	}
	writeToArea(fmt.Sprintf("HP#%v#%%", p.Body[0]), client.Area)
}

// Handles RT#%
func pktWTCE(client *Client, p *packet.Packet) {
	if client.Uid == -1 {
		return
	}
	writeToArea(fmt.Sprintf("RT#%v#%%", p.Body[0]), client.Area)
}

// Handles CT#%
func pktOOC(client *Client, p *packet.Packet) {
	if strings.TrimSpace(p.Body[0]) == "" {
		client.sendServerMessage("You can't send a message without a username!")
		return
	} else if len(p.Body[1]) > config.MaxMsg {
		client.sendServerMessage("Your message exceeds the maximum message length!")
		return
	}

	writeToArea(fmt.Sprintf("CT#%v#%v#0#%%", p.Body[0], p.Body[1]), client.Area)
}

// Handles PE#%
func pktAddEvi(client *Client, p *packet.Packet) {
	client.Area.AddEvidence(strings.Join(p.Body, "&"))
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.Area.GetEvidence(), "#")), client.Area)
}

// Handles DE#%
func pktRemoveEvi(client *Client, p *packet.Packet) {
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.Area.RemoveEvidence(id)
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.Area.GetEvidence(), "#")), client.Area)
}

// Handles EE#%
func pktEditEvi(client *Client, p *packet.Packet) {
	id, err := strconv.Atoi(p.Body[0])
	if err != nil {
		return
	}
	client.Area.EditEvidence(id, strings.Join(p.Body[1:], "&"))
	writeToArea(fmt.Sprintf("LE#%v#%%", strings.Join(client.Area.GetEvidence(), "#")), client.Area)
}

// Handles CH#%
func pktPing(client *Client, _ *packet.Packet) {
	fmt.Fprint(client.Conn, "CHECK#%")
}
