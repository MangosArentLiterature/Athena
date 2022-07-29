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
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/packet"
)

type Client struct {
	mu      sync.Mutex
	conn    net.Conn
	hdid    string
	version string
	uid     int
	area    *area.Area
	char    int
	ipid    string
	oocName string
	lastmsg string
}

// Returns a new client
func newClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
		uid:  -1,
		char: -1,
	}
}

// Handle client handles a client connection to the server
func (client *Client) handleClient() {
	defer client.clientCleanup()
	go timeout(client)

	clients.AddClient(client)

	// For privacy and ease of use, AO servers traditionally use a hashed version of a client's IP address to identify a client.
	// Athena uses the MD5 hash of the IP address, encoded in base64.
	addr := strings.Split(client.conn.RemoteAddr().String(), ":")
	hash := md5.Sum([]byte(strings.Join(addr[:len(addr)-1], ":")))
	client.ipid = base64.StdEncoding.EncodeToString(hash[:])
	client.ipid = client.ipid[:len(client.ipid)-2] // Removes the trailing padding.

	logger.LogDebugf("%v connected", client.ipid)

	client.write("decryptor#NOENCRYPT#%") // Relic of FantaCrypt. AO2 requires a server to send this to proceed with the handshake.
	input := bufio.NewScanner(client.conn)

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
		if logger.DebugNetwork {
			logger.LogDebugf("From %v: %v", client.ipid, strings.TrimSpace(input.Text()))
		}
		packet, err := packet.NewPacket(strings.TrimSpace(input.Text()))
		if err != nil {
			continue // Discard invalid packets
		}
		v := PacketMap[packet.Header] // Check if this is a known packet.
		if v.Func != nil && len(packet.Body) >= v.Args {
			if v.MustJoin && client.uid == -1 {
				return
			}
			v.Func(client, packet)
		}
	}
	logger.LogDebugf("%v disconnected", client.ipid)
}

func (client *Client) write(message string) {
	client.mu.Lock()
	fmt.Fprint(client.conn, message)
	if logger.DebugNetwork {
		logger.LogDebugf("To %v: %v", client.ipid, message)
	}
	client.mu.Unlock()
}

// Cleans up a disconnected client
func (client *Client) clientCleanup() {
	if client.uid != -1 {
		logger.LogInfof("Client (IPID:%v UID:%v) left the server", client.ipid, client.uid)
		uids.ReleaseUid(client.uid)
		players.RemovePlayer()
		client.area.RemoveChar(client.char)
		sendPlayerArup()
	}
	client.conn.Close()
	clients.RemoveClient(client)
}

func (client *Client) sendServerMessage(message string) {
	client.write(fmt.Sprintf("CT#%v#%v#1#%%", config.Name, message))
}

func timeout(client *Client) {
	time.Sleep(1 * time.Minute)
	if client.uid == -1 {
		client.conn.Close()
	}
}
