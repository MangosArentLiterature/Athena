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

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/ms"
	"github.com/MangosArentLiterature/Athena/internal/playercount"
	"github.com/MangosArentLiterature/Athena/internal/settings"
	"github.com/MangosArentLiterature/Athena/internal/uidmanager"
)

const version = "0.1.0"

var (
	config            *settings.Config
	characters, music []string
	areas             []*area.Area
	areaNames         string
	uids              uidmanager.UidManager
	players           playercount.PlayerCount
	clients           ClientList = ClientList{list: make(map[*Client]struct{})}
	updatePlayers                = make(chan int)
	advertDone                   = make(chan struct{})
	FatalError                   = make(chan error)
)

func InitServer(conf *settings.Config) error {
	db.Open()
	uids.InitHeap(conf.MaxPlayers)
	config = conf

	var err error
	music, err = settings.LoadMusic()
	if err != nil {
		return err
	}
	characters, err = settings.LoadCharacters()
	if err != nil {
		return err
	}
	areaData, err := settings.LoadAreas()
	if err != nil {
		return err
	}

	for _, a := range areaData {
		areaNames += a.Name + "#"
		areas = append(areas, area.NewArea(a, len(characters), conf.BufSize))
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

// Starts the server's TCP listener.
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
		client := newClient(conn)
		go client.handleClient()
	}
}

// Sends a message to all connected clients.
func writeToAll(message string) {
	for client := range clients.GetClients() {
		client.write(message)
	}
}

// Sends a message to all clients in an area.
func writeToArea(message string, area *area.Area) {
	for client := range clients.GetClients() {
		if client.area == area {
			client.write(message)
		}
	}
}

// Sends a player ARUP to all clients.
func sendPlayerArup() {
	var plCounts []string
	for _, a := range areas {
		s := strconv.Itoa(a.GetPlayerCount())
		plCounts = append(plCounts, s)
	}
	writeToAll(fmt.Sprintf("ARUP#0#%v#%%", strings.Join(plCounts, "#")))
}

func CleanupServer() {
	for client := range clients.GetClients() {
		client.conn.Close()
	}
	db.Close()
}
