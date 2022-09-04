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

package area

import (
	"strings"
	"sync"

	"github.com/MangosArentLiterature/Athena/internal/sliceutil"
)

type EvidenceMode int
type Status int
type Lock int
type TRState int

const (
	EviMods EvidenceMode = iota
	EviAny
	EviCMs
)
const (
	StatusIdle Status = iota
	StatusPlayers
	StatusCasing
	StatusRecess
	StatusRP
	StatusGaming
)
const (
	LockFree Lock = iota
	LockSpectatable
	LockLocked
)

const (
	TRIdle TRState = iota
	TRRecording
	TRPlayback
	TRUpdating
	TRInserting
)

type TestimonyRecorder struct {
	Testimony []string
	Index     int
	State     TRState
}

type Area struct {
	data     AreaData
	defaults defaults
	mu       sync.Mutex
	taken    []bool
	players  int
	defhp    int
	prohp    int
	evidence []string
	buffer   []string
	cms      []int
	last_msg int
	evi_mode EvidenceMode
	status   Status
	lock     Lock
	invited  []int
	doc      string
	tr       TestimonyRecorder
}

type AreaData struct {
	Name          string `toml:"name"`
	Evi_mode      string `toml:"evidence_mode"`
	Allow_iniswap bool   `toml:"allow_iniswap"`
	Force_noint   bool   `toml:"force_nointerrupt"`
	Bg            string `toml:"background"`
	Allow_cms     bool   `toml:"allow_cms"`
	Force_bglist  bool   `toml:"force_bglist"`
	Lock_bg       bool   `toml:"lock_bg"`
	Lock_music    bool   `toml:"lock_music"`
}

type defaults struct {
	evi_mode      EvidenceMode
	allow_iniswap bool
	force_noint   bool
	bg            string
	allow_cms     bool
	force_bglist  bool
	lock_bg       bool
	lock_music    bool
}

// NewArea returns a new area.
func NewArea(data AreaData, charlen int, bufsize int, evi_mode EvidenceMode) *Area {
	return &Area{
		data: data,
		defaults: defaults{
			evi_mode:      evi_mode,
			allow_iniswap: data.Allow_iniswap,
			force_noint:   data.Force_noint,
			bg:            data.Bg,
			allow_cms:     data.Allow_cms,
			force_bglist:  data.Force_bglist,
			lock_bg:       data.Lock_bg,
			lock_music:    data.Lock_music,
		},
		taken:    make([]bool, charlen),
		defhp:    10,
		prohp:    10,
		buffer:   make([]string, bufsize),
		last_msg: -1,
		evi_mode: evi_mode,
	}
}

// Name returns the area's name.
func (a *Area) Name() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Name
}

// Taken returns the area's taken list, where "-1" is taken and "0" is free
func (a *Area) Taken() []string {
	a.mu.Lock()
	var takenList []string
	for _, t := range a.taken {
		if t {
			takenList = append(takenList, "-1")
		} else {
			takenList = append(takenList, "0")
		}
	}
	a.mu.Unlock()
	return takenList
}

// AddChar adds a new player to the area.
func (a *Area) AddChar(char int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if char != -1 {
		if a.taken[char] {
			return false
		} else {
			a.taken[char] = true
		}
	}
	a.players++
	return true
}

// SwitchChar switches a player's character.
func (a *Area) SwitchChar(old int, new int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if new == -1 {
		if old != -1 {
			a.taken[old] = false
		}
		return true
	} else {
		if a.taken[new] {
			return false
		} else {
			a.taken[new] = true
			if old != -1 {
				a.taken[old] = false
			}
		}
		return true
	}
}

// RemoveChar removes a player from the area.
func (a *Area) RemoveChar(char int) {
	a.mu.Lock()
	if char != -1 {
		a.taken[char] = false
	}
	a.players--
	a.mu.Unlock()
}

// HP returns the values of the area's def and pro HP bars.
func (a *Area) HP() (int, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.defhp, a.prohp
}

// SetHP sets either the def or pro HP to the specified value.
// The bar must be 1 for the defense HP, 2 for pro HP.
// The value must be between 0 and 10.
func (a *Area) SetHP(bar int, v int) bool {
	if v > 10 || v < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	switch bar {
	case 1:
		a.defhp = v
	case 2:
		a.prohp = v
	default:
		return false
	}
	return true
}

// PlayerCount returns the number of players in the area.
func (a *Area) PlayerCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.players
}

// Evidence returns a list of evidence in the area.
func (a *Area) Evidence() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.evidence
}

// AddEvidence adds a piece of evidence to the area.
func (a *Area) AddEvidence(evi string) {
	a.mu.Lock()
	a.evidence = append(a.evidence, evi)
	a.mu.Unlock()
}

// RemoveEvidence removes a piece of evidence to the area.
func (a *Area) RemoveEvidence(id int) {
	a.mu.Lock()
	if len(a.evidence) >= id {
		copy(a.evidence[id:], a.evidence[id+1:])
		a.evidence = a.evidence[:len(a.evidence)-1]
	}
	a.mu.Unlock()
}

// EditEvidence replaces a piece of evidence.
func (a *Area) EditEvidence(id int, evi string) {
	a.mu.Lock()
	if len(a.evidence) >= id {
		a.evidence[id] = evi
	}
	a.mu.Unlock()
}

// SwapEvidence swaps the indexes of two pieces of evidence.
func (a *Area) SwapEvidence(x int, y int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.evidence) < x+1 || len(a.evidence) < y+1 {
		return false
	}
	a.evidence[x], a.evidence[y] = a.evidence[y], a.evidence[x]
	return true
}

// UpdateBuffer adds a new line to the area's log buffer.
func (a *Area) UpdateBuffer(s string) {
	a.mu.Lock()
	a.buffer = append(a.buffer[1:], s)
	a.mu.Unlock()
}

// Buffer returns the area's log buffer.
func (a *Area) Buffer() []string {
	var returnList []string
	a.mu.Lock()
	for _, s := range a.buffer {
		if strings.TrimSpace(s) != "" {
			returnList = append(returnList, s)
		}
	}
	a.mu.Unlock()
	return returnList
}

// CMs returns the list uids of CMs in the area.
func (a *Area) CMs() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cms
}

// AddCM adds a new CM to the area.
func (a *Area) AddCM(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if sliceutil.ContainsInt(a.cms, uid) {
		return false
	}
	a.cms = append(a.cms, uid)
	return true
}

// RemoveCM removes a CM from the area.
func (a *Area) RemoveCM(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, id := range a.cms {
		if id == uid {
			a.cms = append(a.cms[:i], a.cms[i+1:]...)
			return true
		}
	}
	return false
}

// HasCM returns whether the given uid is a CM in the area.
func (a *Area) HasCM(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return sliceutil.ContainsInt(a.cms, uid)
}

// EvidenceMode returns the area's evidence mode.
func (a *Area) EvidenceMode() EvidenceMode {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.evi_mode
}

// SetEvidenceMode sets the area's evidence mode.
func (a *Area) SetEvidenceMode(mode EvidenceMode) {
	a.mu.Lock()
	a.evi_mode = mode
	a.mu.Unlock()
}

// IniswapAllowed returns whether iniswapping is allowed in the area.
func (a *Area) IniswapAllowed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Allow_iniswap
}

// SetIniswapAllowed sets iniswapping in the area.
func (a *Area) SetIniswapAllowed(b bool) {
	a.mu.Lock()
	a.data.Allow_iniswap = b
	a.mu.Unlock()
}

// NoInterrupt returns whether preanims must not interrupt in the area.
func (a *Area) NoInterrupt() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Force_noint
}

// SetNoInterrupt sets non-interrupting preanims in the area.
func (a *Area) SetNoInterrupt(b bool) {
	a.mu.Lock()
	a.data.Force_noint = b
	a.mu.Unlock()
}

// LastSpeaker returns the character of the the last speaker.
func (a *Area) LastSpeaker() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last_msg
}

// SetLastSpeaker sets the area's last speaker.
func (a *Area) SetLastSpeaker(char int) {
	a.mu.Lock()
	a.last_msg = char
	a.mu.Unlock()
}

// Background returns the area's current background.
func (a *Area) Background() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Bg
}

// SetBackground sets the area's background.
func (a *Area) SetBackground(bg string) {
	a.mu.Lock()
	a.data.Bg = bg
	a.mu.Unlock()
}

// IsTaken returns whether the given character is taken in the area.
func (a *Area) IsTaken(char int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if char != -1 {
		return a.taken[char]
	} else {
		return false
	}
}

// CMsAllowed returns whether CMs are allowed in the area.
func (a *Area) CMsAllowed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Allow_cms
}

// SetCMsAllowed sets allowing CMs in the area.
func (a *Area) SetCMsAllowed(b bool) {
	a.mu.Lock()
	a.data.Allow_cms = b
	a.mu.Unlock()
}

// Status returns the area's current status.
func (a *Area) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

// SetStatus sets the area's status.
func (a *Area) SetStatus(status Status) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}

// Lock returns the area's lock type.
func (a *Area) Lock() Lock {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lock
}

// SetLock sets the area's lock.
func (a *Area) SetLock(lock Lock) {
	a.mu.Lock()
	a.lock = lock
	a.mu.Unlock()
}

// AddInvited adds a new UID to the area's invite list.
func (a *Area) AddInvited(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if sliceutil.ContainsInt(a.invited, uid) {
		return false
	}
	a.invited = append(a.invited, uid)
	return true
}

// RemoveInvited removes a UID from the area's invite list.
func (a *Area) RemoveInvited(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, id := range a.invited {
		if id == uid {
			a.invited = append(a.invited[:i], a.invited[i+1:]...)
			return true
		}
	}
	return false
}

// ClearInvited clears the area's invite list.
func (a *Area) ClearInvited() {
	a.mu.Lock()
	a.invited = []int{}
	a.mu.Unlock()
}

// Invited returns the area's invite list.
func (a *Area) Invited() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.invited
}

// Reset returns all area settings to their default values.
func (a *Area) Reset() {
	a.mu.Lock()
	a.evidence = []string{}
	a.invited = []int{}
	a.status = StatusIdle
	a.lock = LockFree
	a.cms = []int{}
	a.last_msg = -1
	a.defhp = 10
	a.prohp = 10
	a.evi_mode = a.defaults.evi_mode
	a.data.Allow_cms = a.defaults.allow_cms
	a.data.Allow_iniswap = a.defaults.allow_iniswap
	a.data.Force_noint = a.defaults.force_noint
	a.data.Bg = a.defaults.bg
	a.data.Force_bglist = a.defaults.force_bglist
	a.data.Lock_bg = a.defaults.lock_bg
	a.data.Lock_music = a.defaults.lock_music
	a.tr.Index = 0
	a.tr.State = TRIdle
	a.tr.Testimony = []string{}
	a.mu.Unlock()
}

// ForceBGList returns whether the server BG list is enforced in the area.
func (a *Area) ForceBGList() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Force_bglist
}

// SetForceBGList sets enforciing the server BG list in the area.
func (a *Area) SetForceBGList(b bool) {
	a.mu.Lock()
	a.data.Force_bglist = b
	a.mu.Unlock()
}

// LockBG returns whether the BG is locked in the area.
func (a *Area) LockBG() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Lock_bg
}

// SetLockBG sets locking the area's BG.
func (a *Area) SetLockBG(b bool) {
	a.mu.Lock()
	a.data.Lock_bg = b
	a.mu.Unlock()
}

// LockMusic returns whether music in the area is CM-only.
func (a *Area) LockMusic() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Lock_music
}

// SetLockMusic sets CM-only music in the area.
func (a *Area) SetLockMusic(b bool) {
	a.mu.Lock()
	a.data.Lock_music = b
	a.mu.Unlock()
}

// Doc returns the area's doc.
func (a *Area) Doc() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.doc
}

// SetDoc sets the area's doc.
func (a *Area) SetDoc(s string) {
	a.mu.Lock()
	a.doc = s
	a.mu.Unlock()
}

// HasTestimony returns whether the area has a recorded testimony.
func (a *Area) HasTestimony() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.tr.Testimony) > 2
}

// Testimony returns the area's recorded testimony.
func (a *Area) Testimony() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var rl []string
	for i, s := range a.tr.Testimony {
		if i == 0 {
			continue
		}
		rl = append(rl, strings.Split(s, "#")[4])
	}
	return rl
}

// String returns the string representation of the status.
func (status Status) String() string {
	switch status {
	case StatusIdle:
		return "IDLE"
	case StatusPlayers:
		return "LOOKING-FOR-PLAYERS"
	case StatusCasing:
		return "CASING"
	case StatusRecess:
		return "RECESS"
	case StatusRP:
		return "RP"
	case StatusGaming:
		return "GAMING"
	}
	return ""
}

// String returns the string representation of the lock.
func (lock Lock) String() string {
	switch lock {
	case LockFree:
		return "FREE"
	case LockSpectatable:
		return "SPECTATABLE"
	case LockLocked:
		return "LOCKED"
	}
	return ""
}

// String returns the string representation of the evimod.
func (evimod EvidenceMode) String() string {
	switch evimod {
	case EviAny:
		return "any"
	case EviCMs:
		return "cms"
	case EviMods:
		return "mods"
	}
	return ""
}
