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
	"container/heap"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/ms"
	"github.com/MangosArentLiterature/Athena/internal/settings"
	"github.com/MangosArentLiterature/Athena/internal/uidheap"
)

const version = "0.1.0"

var verbose bool
var config *settings.Config
var playerCountMu sync.Mutex
var playerCount int
var clientsMu sync.Mutex
var clients = make(map[*Client]struct{})
var characters []string
var music []string
var areas []*area.Area
var areaNames string
var uidsMu sync.Mutex
var uids uidheap.UidHeap
var updatePlayers = make(chan int)
var advertDone = make(chan struct{})

func InitServer(conf *settings.Config, setVerbose bool) {
	uids = make(uidheap.UidHeap, conf.MaxPlayers)
	for i := range uids {
		uids[i] = i
	}
	heap.Init(&uids)
	config = conf
	verbose = setVerbose

	var err error
	music, err = settings.LoadMusic()
	if err != nil {
		log.Fatalf("athena: while loading music: %v\n", err)
	}
	characters, err = settings.LoadCharacters()
	if err != nil {
		log.Fatalf("athena: while loading characters: %v\n", err)
	}
	areaData, err := settings.LoadAreas()
	if err != nil {
		log.Fatalf("athena: while loading areas: %v\n", err)
	}

	for _, a := range areaData {
		areaNames += a.Name + "#"
		areas = append(areas, area.NewArea(a, len(characters)))
	}
	areaNames = strings.TrimSuffix(areaNames, "#")

	if config.Advertise {
		playerCountMu.Lock()
		advert := ms.Advertisement{
			Port:    config.Port,
			Players: playerCount,
			Name:    config.Name,
			Desc:    config.Desc}
		playerCountMu.Unlock()
		go ms.Advertise(config.MSAddr, advert, updatePlayers, advertDone)
	}
}

// Starts the server's TCP listener.
func ListenTCP() {
	listener, err := net.Listen("tcp", config.Addr+":"+strconv.Itoa(config.Port))

	if err != nil {
		log.Fatalf("athena: failed to start server: %v\n", err)
	}

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("athena: %v", err)
		}
		if verbose {
			log.Printf("athena: connection from %v\n", conn.RemoteAddr().String())
		}
		client := NewClient(conn)
		go client.HandleClient()
	}
}

// Sends a message to all connected clients.
func writeToAll(message string) {
	clientsMu.Lock()
	for client := range clients {
		fmt.Fprint(client.Conn, message)
	}
	clientsMu.Unlock()
}

// Sends a message to all clients in an area.
func writeToArea(message string, area *area.Area) {
	clientsMu.Lock()
	for client := range clients {
		if client.Area == area {
			fmt.Fprint(client.Conn, message)
		}
	}
	clientsMu.Unlock()
}

// Sends a player ARUP to all clients.
func sendPlayerArup() {
	var plCounts []string
	for _, a := range areas {
		s := strconv.Itoa(a.GetPlayers())
		plCounts = append(plCounts, s)
	}
	writeToAll(fmt.Sprintf("ARUP#0#%v#%%", strings.Join(plCounts, "#")))
}

// Returns the server's player count.
func getPlayerCount() int {
	playerCountMu.Lock()
	defer playerCountMu.Unlock()
	return playerCount
}
