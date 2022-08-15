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
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/ms"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
	"github.com/MangosArentLiterature/Athena/internal/playercount"
	"github.com/MangosArentLiterature/Athena/internal/settings"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
	"github.com/MangosArentLiterature/Athena/internal/uidmanager"
	"github.com/xhit/go-str2duration/v2"
)

const version = ""

var (
	config                         *settings.Config
	characters, music, backgrounds []string
	areas                          []*area.Area
	areaNames                      string
	roles                          []permissions.Role
	uids                           uidmanager.UidManager
	players                        playercount.PlayerCount
	clients                        ClientList = ClientList{list: make(map[*Client]struct{})}
	updatePlayers                             = make(chan int)
	advertDone                                = make(chan struct{})
	FatalError                                = make(chan error)
)

// InitServer initalizes the server's database, uids, configs, and advertiser.
func InitServer(conf *settings.Config) error {
	db.Open()
	uids.InitHeap(conf.MaxPlayers)
	config = conf

	var err error
	music, err = settings.LoadMusic()
	if err != nil {
		return err
	}
	characters, err = settings.LoadFile("/characters.txt")
	if err != nil {
		return err
	} else if len(characters) == 0 {
		return fmt.Errorf("empty character list")
	}
	areaData, err := settings.LoadAreas()
	if err != nil {
		return err
	}

	roles, err = settings.LoadRoles()
	if err != nil {
		return err
	}

	backgrounds, err = settings.LoadFile("/backgrounds.txt")
	if err != nil {
		return err
	} else if len(backgrounds) == 0 {
		return fmt.Errorf("empty background list")
	}

	_, err = str2duration.ParseDuration(conf.BanLen)
	if err != nil {
		return fmt.Errorf("failed to parse default_ban_duration: %v", err.Error())
	}

	for _, a := range areaData {
		areaNames += a.Name + "#"
		var evi_mode area.EvidenceMode
		switch strings.ToLower(a.Evi_mode) {
		case "any":
			evi_mode = area.EviAny
		case "cms":
			evi_mode = area.EviCMs
		case "none":
			evi_mode = area.EviNone
		default:
			logger.LogWarningf("Area %v has an invalid or undefined evidence mode, defaulting to 'cms'.", a.Name)
			evi_mode = area.EviCMs
		}
		if a.Bg == "" || !sliceutil.ContainsString(backgrounds, a.Bg) {
			logger.LogWarningf("Area %v has an invalid or undefined background, defaulting to 'default'.", a.Name)
			a.Bg = "default"
		}
		areas = append(areas, area.NewArea(a, len(characters), conf.BufSize, evi_mode))
	}
	areaNames = strings.TrimSuffix(areaNames, "#")
	if config.Advertise {
		advert := ms.Advertisement{
			Port:    config.Port,
			Players: players.GetPlayerCount(),
			Name:    config.Name,
			Desc:    config.Desc}
		go ms.Advertise(config.MSAddr, advert, updatePlayers, advertDone)
	}
	return nil
}

// ListenTCP starts the server's TCP listener.
func ListenTCP() {
	listener, err := net.Listen("tcp", config.Addr+":"+strconv.Itoa(config.Port))

	if err != nil {
		FatalError <- err
		return
	}

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.LogWarning(err.Error())
		}
		if logger.DebugNetwork {
			logger.LogDebugf("Connection recieved from %v", conn.RemoteAddr())
		}
		client := NewClient(conn)
		go client.HandleClient()
	}
}

// writeToAll sends a message to all connected clients.
func writeToAll(header string, contents ...string) {
	for client := range clients.GetAllClients() {
		client.SendPacket(header, contents...)
	}
}

// writeToArea sends a message to all clients in a given area.
func writeToArea(area *area.Area, header string, contents ...string) {
	for client := range clients.GetAllClients() {
		if client.Area() == area {
			client.SendPacket(header, contents...)
		}
	}
}

// addToBuffer writes to an area buffer according to a client's action.
func addToBuffer(client *Client, action string, message string, audit bool) {
	var auth string
	if client.Authenticated() {
		auth = " (*)"
	}
	s := fmt.Sprintf("[%v] [%v] %v%v (%v) %v: %v", time.Now().Format("15:04:05"), action,
		client.CurrentCharacter(), auth, client.Ipid(), client.OOCName(), message)
	client.Area().UpdateBuffer(s)
	if audit {
		logger.WriteAudit(s)
	}
}

// sendPlayerArup sends a player ARUP update to all connected clients.
func sendPlayerArup() {
	var plCounts []string
	for _, a := range areas {
		s := strconv.Itoa(a.PlayerCount())
		plCounts = append(plCounts, s)
	}
	writeToAll(fmt.Sprintf("ARUP#0#%v#%%", strings.Join(plCounts, "#")))
}

// getRole returns the role with the corresponding name, or an error if the role does not exist.
func getRole(name string) (permissions.Role, error) {
	for _, role := range roles {
		if role.Name == name {
			return role, nil
		}
	}
	return permissions.Role{}, fmt.Errorf("role does not exist")
}

// getClientByUid returns the client with the given uid.
func getClientByUid(uid int) (*Client, error) {
	for c := range clients.GetAllClients() {
		if c.Uid() == uid {
			return c, nil
		}
	}
	return nil, fmt.Errorf("client does not exist")
}

// getClientsByIpid returns all clients with the given ipid.
func getClientsByIpid(ipid string) []*Client {
	var returnlist []*Client
	for c := range clients.GetAllClients() {
		if c.Ipid() == ipid {
			returnlist = append(returnlist, c)
		}
	}
	return returnlist
}

// sendAreaServerMessage sends a server OOC message to all clients in an area.
func sendAreaServerMessage(area *area.Area, message string) {
	writeToArea(area, "CT", encode(config.Name), encode(message), "1")
}

// CleanupServer closes all connections to the server, and closes the server's database.
func CleanupServer() {
	for client := range clients.GetAllClients() {
		client.conn.Close()
	}
	db.Close()
}
