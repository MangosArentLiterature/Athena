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
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MangosArentLiterature/Athena/internal/area"
	"github.com/MangosArentLiterature/Athena/internal/db"
	"github.com/MangosArentLiterature/Athena/internal/logger"
	"github.com/MangosArentLiterature/Athena/internal/packet"
	"github.com/MangosArentLiterature/Athena/internal/permissions"
	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
	"go.uber.org/ratelimit"
)

type MuteState int

const (
	Unmuted MuteState = iota
	ICMuted
	OOCMuted
	ICOOCMuted
	MusicMuted
	JudMuted
	ParrotMuted
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
	muted         MuteState
	muteuntil     time.Time
	showname      string
}

// NewClient returns a new client.
func NewClient(conn net.Conn, ipid string) *Client {
	return &Client{
		conn: conn,
		uid:  -1,
		char: -1,
		pair: ClientPairInfo{wanted_id: -1},
		ipid: ipid,
	}
}

// handleClient handles a client connection to the server.
func (client *Client) HandleClient() {
	defer client.clientCleanup()

	client.CheckBanned(db.IPID)

	var mc int
	for c := range clients.GetAllClients() {
		if c.Ipid() == client.Ipid() {
			mc++
		}
	}
	if mc >= config.MCLimit && config.MCLimit != 0 {
		client.SendPacket("BD", "You have reached the server's multiclient limit.")
		client.conn.Close()
		return
	}

	logger.LogDebugf("%v connected", client.ipid)
	clients.AddClient(client)

	go timeout(client)

	client.SendPacket("decryptor", "NOENCRYPT") // Relic of FantaCrypt. AO2 requires a server to send this to proceed with the handshake.
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

	rl := ratelimit.New(10, ratelimit.WithoutSlack)
	for input.Scan() {
		rl.Take()
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

// write sends the given message to the client's network socket.
func (client *Client) write(message string) {
	client.mu.Lock()
	fmt.Fprint(client.conn, message)
	if logger.DebugNetwork {
		logger.LogDebugf("To %v: %v", client.ipid, message)
	}
	client.mu.Unlock()
}

// SendPacket sends the client a packet with the given header and contents.
func (client *Client) SendPacket(header string, contents ...string) {
	client.write(header + "#" + strings.Join(contents, "#") + "#%")
}

// clientClenup cleans up a disconnected client.
func (client *Client) clientCleanup() {
	if client.Uid() != -1 {
		logger.LogInfof("Client (IPID:%v UID:%v) left the server", client.ipid, client.Uid())

		if client.Area().PlayerCount() <= 1 {
			client.Area().Reset()
			sendLockArup()
			sendStatusArup()
			sendCMArup()
		} else if client.Area().HasCM(client.Uid()) {
			client.Area().RemoveCM(client.Uid())
			sendCMArup()
		}
		for _, a := range areas {
			if a.Lock() != area.LockFree {
				a.RemoveInvited(client.Uid())
			}
		}
		uids.ReleaseUid(client.Uid())
		players.RemovePlayer()
		updatePlayers <- players.GetPlayerCount()
		client.Area().RemoveChar(client.CharID())
		sendPlayerArup()
	}
	client.conn.Close()
	clients.RemoveClient(client)
}

// SendServerMessage sends a server OOC message to the client.
func (client *Client) SendServerMessage(message string) {
	client.SendPacket("CT", encode(config.Name), encode(message), "1")
}

// CurrentCharacter returns the client's current character name.
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

// Hdid returns the client's hdid.
func (client *Client) Hdid() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.hdid
}

// SetHdid sets the client's hdid.
func (client *Client) SetHdid(hdid string) {
	client.mu.Lock()
	client.hdid = hdid
	client.mu.Unlock()
}

// Uid returns the client's user ID.
func (client *Client) Uid() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.uid
}

// SetUid sets the client's user ID.
func (client *Client) SetUid(id int) {
	client.mu.Lock()
	client.uid = id
	client.mu.Unlock()
}

// Area returns the client's current area.
func (client *Client) Area() *area.Area {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.area
}

// SetArea sets the client's current area.
func (client *Client) SetArea(area *area.Area) {
	client.mu.Lock()
	client.area = area
	client.mu.Unlock()
}

// CharID returns the client's character ID.
func (client *Client) CharID() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.char
}

// SetCharID sets the client's character ID.
func (client *Client) SetCharID(id int) {
	client.mu.Lock()
	client.char = id
	client.mu.Unlock()
}

// Ipid returns the client's ipid.
func (client *Client) Ipid() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.ipid
}

// OOCName returns the client's current OOC username.
func (client *Client) OOCName() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.oocName
}

// SetOocName sets the client's OOC username.
func (client *Client) SetOocName(name string) {
	client.mu.Lock()
	client.oocName = name
	client.mu.Unlock()
}

// LastMsg returns the client's last sent IC message.
func (client *Client) LastMsg() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.lastmsg
}

// SetLastMsg sets the client's last sent IC message.
func (client *Client) SetLastMsg(msg string) {
	client.mu.Lock()
	client.lastmsg = msg
	client.mu.Unlock()
}

// Perms returns the client's current permissions.
func (client *Client) Perms() uint64 {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.perms
}

// SetPerms sets the client's permissionss.
func (client *Client) SetPerms(perms uint64) {
	client.mu.Lock()
	client.perms = perms
	client.mu.Unlock()
}

// Authenticated returns whether the client is logged in as a moderator.
func (client *Client) Authenticated() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.authenticated
}

// SetAuthenticated sets whether the client is logged in as a moderator.
func (client *Client) SetAuthenticated(auth bool) {
	client.mu.Lock()
	client.authenticated = auth
	client.mu.Unlock()
}

// ModName returns the client's moderator username.
func (client *Client) ModName() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.mod_name
}

// SetModName sets the client's moderator username.
func (client *Client) SetModName(name string) {
	client.mu.Lock()
	client.mod_name = name
	client.mu.Unlock()
}

// Pos returns the client's current position.
func (client *Client) Pos() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pos
}

// SetPos sets the client's position.
func (client *Client) SetPos(pos string) {
	client.mu.Lock()
	client.pos = pos
	client.mu.Unlock()
}

// CasePrefs returns all client's case preferences.
func (client *Client) CasePrefs() [5]bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.case_prefs
}

// CasePref returns a client's role alert preference.
func (client *Client) AlertRole(index int) bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.case_prefs[index]
}

// SetCasePref sets a client's role alert preference.
func (client *Client) SetRoleAlert(index int, b bool) {
	client.mu.Lock()
	client.case_prefs[index] = b
	client.mu.Unlock()
}

// PairInfo returns a client's pairing info.
func (client *Client) PairInfo() ClientPairInfo {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pair
}

// SetPairInfo updates a client's pairing info.
func (client *Client) SetPairInfo(name string, emote string, flip string, offset string) {
	client.mu.Lock()
	client.pair.name, client.pair.emote, client.pair.flip, client.pair.offset = name, emote, flip, offset
	client.mu.Unlock()
}

// PairWantedID returns the character the client wishes to pair with.
func (client *Client) PairWantedID() int {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.pair.wanted_id
}

// SetPairWantedID sets the character the client wishes to pair with.
func (client *Client) SetPairWantedID(id int) {
	client.mu.Lock()
	client.pair.wanted_id = id
	client.mu.Unlock()
}

// RemoveAuth logs a client out as moderator.
func (client *Client) RemoveAuth() {
	client.mu.Lock()
	client.authenticated, client.perms, client.mod_name = false, 0, ""
	client.mu.Unlock()
	client.SendServerMessage("Logged out as moderator.")
	client.SendPacket("AUTH", "-1")
}

// CheckBanned returns if a client is currently banned.
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
		client.SendPacket("BD", fmt.Sprintf("%v\nUntil: %v\nID: %v", baninfo.Reason, duration, baninfo.Id))
		client.conn.Close()
		return
	}
}

// JoinArea adds a client to an area.
func (client *Client) JoinArea(area *area.Area) {
	client.SetArea(area)
	area.AddChar(client.CharID())
	def, pro := area.HP()
	client.SendPacket("LE", areas[0].Evidence()...)
	client.SendPacket("CharsCheck", area.Taken()...)
	client.SendPacket("HP", "1", strconv.Itoa(def))
	client.SendPacket("HP", "2", strconv.Itoa(pro))
	client.SendPacket("BN", area.Background())
	sendPlayerArup()
}

// ChangeArea changes the client's current area.
func (client *Client) ChangeArea(a *area.Area) bool {
	if a.Lock() == area.LockLocked &&
		!sliceutil.ContainsInt(a.Invited(), client.Uid()) &&
		!permissions.HasPermission(client.Perms(), permissions.PermissionField["BYPASS_LOCK"]) {
		return false
	}
	addToBuffer(client, "AREA", "Left area.", false)
	if client.Area().PlayerCount() <= 1 {
		client.Area().Reset()
		sendLockArup()
		sendStatusArup()
		sendCMArup()
	} else if client.Area().HasCM(client.Uid()) {
		client.Area().RemoveCM(client.Uid())
		sendCMArup()
	}
	client.Area().RemoveChar(client.CharID())
	if a.IsTaken(client.CharID()) {
		client.SetCharID(-1)
	}
	client.JoinArea(a)
	if client.CharID() == -1 {
		client.SendPacket("DONE")
	} else {
		writeToArea(a, "CharsCheck", a.Taken()...)
	}
	addToBuffer(client, "AREA", "Joined area.", false)
	return true
}

// HasCMPermission returns whether the client has CM permissions in it's area.
func (client *Client) HasCMPermission() bool {
	if client.Area().HasCM(client.Uid()) || permissions.HasPermission(client.Perms(), permissions.PermissionField["CM"]) {
		return true
	} else {
		return false
	}
}

// CanSpeakIC returns whether the client can send IC messages.
func (client *Client) CanSpeakIC() bool {
	switch {
	case client.CharID() == -1:
		return false
	case client.Area().Lock() == area.LockSpectatable && !sliceutil.ContainsInt(client.area.Invited(), client.Uid()) &&
		!permissions.HasPermission(client.Perms(), permissions.PermissionField["BYPASS_LOCK"]):
		return false
	case client.Muted() == ICMuted || client.Muted() == ICOOCMuted:
		return client.CheckUnmute()
	}
	return true
}

// CanSpeakOOC returns whether the client can send OOC messages.
func (client *Client) CanSpeakOOC() bool {
	if client.Muted() == OOCMuted || client.Muted() == ICOOCMuted {
		return client.CheckUnmute()
	}
	return true
}

// CanChangeMusic returns whether the client can change the music.
func (client *Client) CanChangeMusic() bool {
	switch {
	case client.CharID() == -1:
		return false
	case client.Area().LockMusic() && !client.HasCMPermission():
		return false
	case client.Area().Lock() == area.LockSpectatable && !sliceutil.ContainsInt(client.area.Invited(), client.Uid()) &&
		!permissions.HasPermission(client.Perms(), permissions.PermissionField["BYPASS_LOCK"]):
		return false
	case client.Muted() == MusicMuted || client.Muted() == ICMuted || client.Muted() == ICOOCMuted:
		return client.CheckUnmute()
	}
	return true
}

// CanJud returns whether the client can use judge actions.
func (client *Client) CanJud() bool {
	switch {
	case client.CharID() == -1:
		return false
	case client.Area().Lock() == area.LockSpectatable && !sliceutil.ContainsInt(client.area.Invited(), client.Uid()) &&
		!permissions.HasPermission(client.Perms(), permissions.PermissionField["BYPASS_LOCK"]):
		return false
	case client.Muted() == JudMuted || client.Muted() == ICMuted || client.Muted() == ICOOCMuted:
		return client.CheckUnmute()
	}
	return true
}

// CheckUnmute checks the client's mute duration, unmuting them if nessecary, and returning whether the client is still muted.
func (client *Client) CheckUnmute() bool {
	if time.Now().UTC().After(client.UnmuteTime()) && !client.UnmuteTime().IsZero() {
		client.SendServerMessage("You have been unmuted.")
		client.SetMuted(Unmuted)
		return true
	}
	return false
}

// IsParrot returns if the client has been parroted.
func (client *Client) IsParrot() bool {
	if client.Muted() == ParrotMuted {
		return !client.CheckUnmute()
	}
	return false
}

// canAlterEvidence is a helper function that returns if a client can alter evidence in their current area.
func (client *Client) CanAlterEvidence() bool {
	if client.CharID() == -1 || !client.CanSpeakIC() {
		return false
	}
	switch client.Area().EvidenceMode() {
	case area.EviMods:
		if !permissions.HasPermission(client.Perms(), permissions.PermissionField["MOD_EVI"]) {
			return false
		}
	case area.EviCMs:
		if !client.HasCMPermission() {
			return false
		}
	}
	return true
}

// ChangeCharacter changes the client's character to the given character.
func (client *Client) ChangeCharacter(id int) {
	if client.Area().SwitchChar(client.CharID(), id) {
		client.SetCharID(id)
		client.SetShowname(characters[id])
		client.SendPacket("PV", "0", "CID", strconv.Itoa(id))
		writeToArea(client.Area(), "CharsCheck", client.Area().Taken()...)
	}
}

// Muted returns the client's mute state.
func (client *Client) Muted() MuteState {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.muted
}

// SetMuted sets the client's mute state.
func (client *Client) SetMuted(m MuteState) {
	client.mu.Lock()
	client.muted = m
	client.mu.Unlock()
}

// UnmuteTime returns the time when the client should be unmuted.
// If this the time is zero, the mute does not expire.
func (client *Client) UnmuteTime() time.Time {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.muteuntil
}

// SetUnmuteTime sets the time when the client should be unmuted.
func (client *Client) SetUnmuteTime(t time.Time) {
	client.mu.Lock()
	client.muteuntil = t
	client.mu.Unlock()
}

// Showname returns the client's showname.
func (client *Client) Showname() string {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.showname
}

// SetShowname sets the client's showname
func (client *Client) SetShowname(s string) {
	client.mu.Lock()
	client.showname = s
	client.mu.Unlock()
}

// String returns the string representation of a mute state.
func (m MuteState) String() string {
	switch m {
	case ICMuted:
		return "IC"
	case OOCMuted:
		return "OOC"
	case ICOOCMuted:
		return "IC/OOC"
	}
	return ""
}
