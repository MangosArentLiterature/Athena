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
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/packet"
)

type ClientPairInfo struct {
	name      string
	emote     string
	flip      string
	offset    string
	wanted_id int
}

type Client struct {
	pair          ClientPairInfo
	mu            sync.Mutex
	conn          net.Conn
	joining       bool
	hdid          string
	uid           int
	area          *area.Area
	char          int
	ipid          string
	oocName       string
	lastmsg       string
	perms         uint64
	authenticated bool
	mod_name      string
	pos           string
	case_prefs    [5]bool
}

// Returns a new client.
func NewClient(conn net.Conn) *Client {
	return &Client{
		conn: conn,
		uid:  -1,
		char: -1,
		pair: ClientPairInfo{wanted_id: -1},
	}
}

// handleClient handles a client connection to the server.
func (client *Client) HandleClient() {
	defer client.clientCleanup()

	// For privacy and ease of use, AO servers traditionally use a hashed version of a client's IP address to identify a client.
	// Athena uses the MD5 hash of the IP address, encoded in base64.
	addr := strings.Split(client.conn.RemoteAddr().String(), ":")
	hash := md5.Sum([]byte(strings.Join(addr[:len(addr)-1], ":")))
	client.ipid = base64.StdEncoding.EncodeToString(hash[:])
	client.ipid = client.ipid[:len(client.ipid)-2] // Removes the trailing padding.

	client.CheckBanned(db.IPID)
	logger.LogDebugf("%v connected", client.ipid)
	clients.AddClient(client)

	go timeout(client)

	client.Write("decryptor#NOENCRYPT#%") // Relic of FantaCrypt. AO2 requires a server to send this to proceed with the handshake.
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
			if v.MustJoin && client.Uid() == -1 {
				continue
			}
			v.Func(client, packet)
		}
	}
	logger.LogDebugf("%v disconnected", client.ipid)
}

// Writes a string to the client's network socket.
func (client *Client) Write(message string) {
	client.mu.Lock()
	fmt.Fprint(client.conn, message)
	if logger.DebugNetwork {
		logger.LogDebugf("To %v: %v", client.ipid, message)
	}
	client.mu.Unlock()
}

// clientClenup cleans up a disconnected client.
func (client *Client) clientCleanup() {
	if client.Uid() != -1 {
		logger.LogInfof("Client (IPID:%v UID:%v) left the server", client.ipid, client.Uid())
		uids.ReleaseUid(client.Uid())
		players.RemovePlayer()
		client.Area().RemoveChar(client.CharID())
		sendPlayerArup()
	}
	client.conn.Close()
	clients.RemoveClient(client)
}

// SendServerMessage sends a server OOC message to the client.
func (client *Client) SendServerMessage(message string) {
	client.Write(fmt.Sprintf("CT#%v#%v#1#%%", encode(config.Name), encode(message)))
}

func (client *Client) CurrentCharacter() string {
	if client.CharID() == -1 {
		return "Spectator"
	} else {
		return characters[client.CharID()]
	}
}

// timeout closes an unjoined client's connection after 1 minute.
func timeout(client *Client) {
	time.Sleep(1 * time.Minute)
	if client.Uid() == -1 {
		client.conn.Close()
	}
}

func (client *Client) Hdid() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.hdid
}

func (client *Client) SetHdid(hdid string) {
	client.mu.Lock()
	client.hdid = hdid
	client.mu.Unlock()
}

func (client *Client) Uid() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.uid
}

func (client *Client) SetUid(id int) {
	client.mu.Lock()
	client.uid = id
	client.mu.Unlock()
}

func (client *Client) Area() *area.Area {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.area
}

func (client *Client) SetArea(area *area.Area) {
	client.mu.Lock()
	client.area = area
	client.mu.Unlock()
}

func (client *Client) CharID() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.char
}

func (client *Client) SetCharID(id int) {
	client.mu.Lock()
	client.char = id
	client.mu.Unlock()
}

func (client *Client) Ipid() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.ipid
}

func (client *Client) OOCName() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.oocName
}

func (client *Client) SetOocName(name string) {
	client.mu.Lock()
	client.oocName = name
	client.mu.Unlock()
}

func (client *Client) LastMsg() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.lastmsg
}

func (client *Client) SetLastMsg(msg string) {
	client.mu.Lock()
	client.lastmsg = msg
	client.mu.Unlock()
}

func (client *Client) Perms() uint64 {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.perms
}

func (client *Client) SetPerms(perms uint64) {
	client.mu.Lock()
	client.perms = perms
	client.mu.Unlock()
}

func (client *Client) Authenticated() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.authenticated
}

func (client *Client) SetAuthenticated(auth bool) {
	client.mu.Lock()
	client.authenticated = auth
	client.mu.Unlock()
}

func (client *Client) ModName() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.mod_name
}

func (client *Client) SetModName(name string) {
	client.mu.Lock()
	client.mod_name = name
	client.mu.Unlock()
}

func (client *Client) Pos() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pos
}

func (client *Client) SetPos(pos string) {
	client.mu.Lock()
	client.pos = pos
	client.mu.Unlock()
}

func (client *Client) CasePrefs() [5]bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.case_prefs
}

func (client *Client) CasePref(index int) bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.case_prefs[index]
}

func (client *Client) SetCasePref(index int, b bool) {
	client.mu.Lock()
	client.case_prefs[index] = b
	client.mu.Unlock()
}

func (client *Client) PairInfo() ClientPairInfo {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pair
}

func (client *Client) SetPairInfo(name string, emote string, flip string, offset string) {
	client.mu.Lock()
	client.pair.name, client.pair.emote, client.pair.flip, client.pair.offset = name, emote, flip, offset
	client.mu.Unlock()
}

func (client *Client) PairWantedID() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pair.wanted_id
}

func (client *Client) SetPairWantedID(id int) {
	client.mu.Lock()
	client.pair.wanted_id = id
	client.mu.Unlock()
}

func (client *Client) RemoveAuth() {
	client.mu.Lock()
	client.authenticated, client.perms, client.mod_name = false, 0, ""
	client.mu.Unlock()
	client.SendServerMessage("Logged out as moderator.")
	client.Write("AUTH#-1#%")
}

func (client *Client) CheckBanned(by db.BanLookup) {
	var banned bool
	var baninfo db.BanInfo
	var err error
	switch by {
	case db.IPID:
		banned, baninfo, err = db.IsBanned(by, client.Ipid())
		if err != nil {
			logger.LogErrorf("Error reading IP ban for %v: %v", client.Ipid(), err)
		}
	case db.HDID:
		banned, baninfo, err = db.IsBanned(by, client.Hdid())
		if err != nil {
			logger.LogErrorf("Error reading HDID ban for %v: %v", client.Ipid(), err)
		}
	}

	if banned {
		var duration string
		if baninfo.Duration == -1 {
			duration = "∞"
		} else {
			duration = time.Unix(baninfo.Duration, 0).UTC().Format("02 Jan 2006 15:04 MST")
		}
		client.Write(fmt.Sprintf("BD#%v\nUntil: %v\nID: %v#%%", baninfo.Reason, duration, baninfo.Id))
		client.conn.Close()
		return
	}
}
