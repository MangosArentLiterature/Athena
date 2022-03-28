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
	"bufio"
	"bytes"
	"container/heap"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/packet"
)

type Client struct {
	Conn    net.Conn
	Hdid    string
	Version string
	Uid     int
	Area    *area.Area
	Char    int
}

// Returns a new client
func NewClient(conn net.Conn) *Client {
	return &Client{
		Conn: conn,
		Uid:  -1,
		Char: -1,
	}
}

// Handle client handles a client connection to the server
func (client *Client) HandleClient() {
	defer client.clientCleanup()
	go timeout(client)

	clientsMu.Lock()
	clients[client] = struct{}{}
	clientsMu.Unlock()

	client.Conn.Write([]byte("decryptor#NOENCRYPT#%")) // Relic of FantaCrypt. AO2 requires this to be sent. Old client versions use NOENCRYPT to disable FantaCrypt.
	input := bufio.NewScanner(client.Conn)
	splitfn := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '%'); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
	input.Split(splitfn) // Split input when a packet delimiter ('%') is found

	for input.Scan() {
		if verbose {
			log.Printf("from %v: %v\n", client.Conn.LocalAddr().String(), strings.TrimSpace(input.Text()))
		}
		packet, err := packet.NewPacket(strings.TrimSpace(input.Text()))
		if err != nil {
			continue // Discard invalid packets
		}
		v := PacketMap[packet.Header]
		if v.Func != nil && len(packet.Body) >= v.Args {
			if v.MustJoin && client.Uid == -1 {
				return
			}
			v.Func(client, packet)
		}
	}
	if verbose {
		log.Printf("%v disconnected\n", client.Conn.RemoteAddr().String())
	}
}

// Cleans up a disconnected client
func (client *Client) clientCleanup() {
	if client.Uid != -1 {
		client.releaseUid()
		playerCountMu.Lock()
		playerCount--
		playerCountMu.Unlock()
		client.Area.Leave(client.Char)
		sendPlayerArup()
	}
	client.Conn.Close()
	clientsMu.Lock()
	delete(clients, client)
	clientsMu.Unlock()
}

func (client *Client) takeUid() {
	uidsMu.Lock()
	client.Uid = heap.Pop(&uids).(int)
	uidsMu.Unlock()
}

func (client *Client) releaseUid() {
	uidsMu.Lock()
	heap.Push(&uids, client.Uid)
	uidsMu.Unlock()
	client.Uid = -1
}

func (client *Client) sendServerMessage(message string) {
	fmt.Fprintf(client.Conn, "CT#%v#%v#1#%%", config.Name, message)
}

func timeout(client *Client) {
	time.Sleep(1 * time.Minute)
	if client.Uid == -1 {
		client.Conn.Close()
	}
}
