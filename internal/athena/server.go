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
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/http"
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
	"github.com/MangosArentLiterature/Athena/internal/webhook"
	"github.com/ecnepsnai/discord"
	"github.com/xhit/go-str2duration/v2"
	"nhooyr.io/websocket"
)

const version = "1.0.1"

var (
	config                                 *settings.Config
	characters, music, backgrounds, parrot []string
	areas                                  []*area.Area
	areaNames                              string
	roles                                  []permissions.Role
	uids                                   uidmanager.UidManager
	players                                playercount.PlayerCount
	enableDiscord                          bool
	clients                                ClientList = ClientList{list: make(map[*Client]struct{})}
	updatePlayers                                     = make(chan int)      // Updates the advertiser's player count.
	advertDone                                        = make(chan struct{}) // Signals the advertiser to stop.
	FatalError                                        = make(chan error)    // Signals that the server should stop after a fatal error.
)

// InitServer initalizes the server's database, uids, configs, and advertiser.
func InitServer(conf *settings.Config) error {
	db.Open()
	uids.InitHeap(conf.MaxPlayers)
	config = conf

	// Load server data.
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

	parrot, err = settings.LoadFile("/parrot.txt")
	if err != nil {
		return err
	} else if len(parrot) == 0 {
		return fmt.Errorf("empty parrot list")
	}
	_, err = str2duration.ParseDuration(conf.BanLen)
	if err != nil {
		return fmt.Errorf("failed to parse default_ban_duration: %v", err.Error())
	}

	// Discord webhook.
	if config.WebhookURL != "" {
		enableDiscord = true
		webhook.ServerName = config.Name
		discord.WebhookURL = config.WebhookURL
	}

	// Load areas.
	for _, a := range areaData {
		areaNames += a.Name + "#"
		var evi_mode area.EvidenceMode
		switch strings.ToLower(a.Evi_mode) {
		case "any":
			evi_mode = area.EviAny
		case "cms":
			evi_mode = area.EviCMs
		case "mods":
			evi_mode = area.EviMods
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
		if config.EnableWS {
			advert.WSPort = config.WSPort
		}
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
	logger.LogDebug("TCP listener started.")
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.LogError(err.Error())
		}
		ipid := getIpid(conn.RemoteAddr().String())
		if logger.DebugNetwork {
			logger.LogDebugf("Connection recieved from %v", ipid)
		}
		client := NewClient(conn, ipid)
		go client.HandleClient()
	}
}

// ListenWS starts the server's websocket listener.
func ListenWS() {
	listener, err := net.Listen("tcp", config.Addr+":"+strconv.Itoa(config.WSPort))
	if err != nil {
		FatalError <- err
		return
	}
	logger.LogDebug("WS listener started.")
	defer listener.Close()

	s := &http.Server{}
	http.HandleFunc("/", HandleWS)
	err = s.Serve(listener)
	if err != http.ErrServerClosed {
		FatalError <- err
	}
}

// HandleWS handles a websocket connection.
func HandleWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"web.aceattorneyonline.com"}}) // WS connections not originating from webAO will be rejected.
	if err != nil {
		logger.LogError(err.Error())
		return
	}
	ipid := getIpid(r.RemoteAddr)
	if logger.DebugNetwork {
		logger.LogDebugf("Connection recieved from %v", ipid)
	}
	client := NewClient(websocket.NetConn(context.TODO(), c, websocket.MessageText), ipid)
	go client.HandleClient()
}

// writeToAll sends a message to all connected clients.
func writeToAll(header string, contents ...string) {
	for client := range clients.GetAllClients() {
		if client.Uid() == -1 {
			continue
		}
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
	s := fmt.Sprintf("%v | %v | %v | %v | %v | %v",
		time.Now().UTC().Format("15:04:05"), action, client.CurrentCharacter(), client.Ipid(), client.OOCName(), message)
	client.Area().UpdateBuffer(s)
	if audit {
		logger.WriteAudit(s)
	}
}

// sendPlayerArup sends a player ARUP to all connected clients.
func sendPlayerArup() {
	plCounts := []string{"0"}
	for _, a := range areas {
		s := strconv.Itoa(a.PlayerCount())
		plCounts = append(plCounts, s)
	}
	writeToAll("ARUP", plCounts...)
}

// sendCMArup sends a CM ARUP to all connected clients.
func sendCMArup() {
	returnL := []string{"2"}
	for _, a := range areas {
		var cms []string
		var uids []int
		uids = append(uids, a.CMs()...)
		if len(uids) == 0 {
			returnL = append(returnL, "FREE")
			continue
		}
		for _, u := range uids {
			c, err := getClientByUid(u)
			if err != nil {
				continue
			}
			cms = append(cms, fmt.Sprintf("%v (%v)", c.CurrentCharacter(), u))
		}
		returnL = append(returnL, strings.Join(cms, ", "))
	}
	writeToAll("ARUP", returnL...)
}

// sendStatusArup sends a status ARUP to all connected clients.
func sendStatusArup() {
	statuses := []string{"1"}
	for _, a := range areas {
		statuses = append(statuses, a.Status().String())
	}
	writeToAll("ARUP", statuses...)
}

// sendLockArup sends a lock ARUP to all connected clients.
func sendLockArup() {
	locks := []string{"3"}
	for _, a := range areas {
		locks = append(locks, a.Lock().String())
	}
	writeToAll("ARUP", locks...)
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

// Returns the IPID for a given IP address.
func getIpid(s string) string {
	// For privacy and ease of use, AO servers traditionally use a hashed version of a client's IP address to identify a client.
	// Athena uses the MD5 hash of the IP address, encoded in base64.
	addr := strings.Split(s, ":")
	hash := md5.Sum([]byte(strings.Join(addr[:len(addr)-1], ":")))
	ipid := base64.StdEncoding.EncodeToString(hash[:])
	return ipid[:len(ipid)-2] // Removes the trailing padding.
}

// getParrotMsg returns a random string from the server's parrot list.
func getParrotMsg() string {
	gen := rand.New(rand.NewSource(time.Now().Unix()))
	return parrot[gen.Intn(len(parrot))]
}
