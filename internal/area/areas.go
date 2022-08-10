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

const (
	EviNone EvidenceMode = iota
	EviAny
	EviCMs
)

type Area struct {
	data     AreaData
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
}

type AreaData struct {
	Name          string `toml:"name"`
	Evi_mode      string `toml:"evidence_mode"`
	Allow_iniswap bool   `toml:"allow_iniswap"`
	Force_noint   bool   `toml:"force_nointerrupt"`
	// bg            string       `toml:"background"`
	// lock_bg       bool         `toml:"lock_bg"`
	// force_bglist  bool         `toml:"enforce_bglist"`
	// lock_music    bool         `toml:"restrict_music"`
}

// Returns a new area
func NewArea(data AreaData, charlen int, bufsize int, evi_mode EvidenceMode) *Area {
	return &Area{
		data:     data,
		taken:    make([]bool, charlen),
		defhp:    10,
		prohp:    10,
		buffer:   make([]string, bufsize),
		last_msg: -1,
		evi_mode: evi_mode,
	}
}

func (a *Area) Name() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Name
}

// Returns the list of taken characters in an area, where "-1" is taken and "0" is free
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

// Adds a player with the specified character to the area. Returns whether the join was successful.
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

// Switches a player's character. Returns whether the switch was successful.
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

// Removes a player with the specified character from the area.
func (a *Area) RemoveChar(char int) {
	a.mu.Lock()
	if char != -1 {
		a.taken[char] = false
	}
	a.players--
	a.mu.Unlock()
}

// Returns the values of the def and pro HP bars.
func (a *Area) HP() (int, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.defhp, a.prohp
}

// Sets either the def or pro HP to the specified value.
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

// Returns the number of players in the area.
func (a *Area) PlayerCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.players
}

// Returns a list of evidence in the area.
func (a *Area) Evidence() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.evidence
}

// Adds the given evidence to the area.
func (a *Area) AddEvidence(evi string) {
	a.mu.Lock()
	a.evidence = append(a.evidence, evi)
	a.mu.Unlock()
}

// Removes the evidence with the given ID.
func (a *Area) RemoveEvidence(id int) {
	a.mu.Lock()
	if len(a.evidence) >= id {
		copy(a.evidence[id:], a.evidence[id+1:])
		a.evidence = a.evidence[:len(a.evidence)-1]
	}
	a.mu.Unlock()
}

// Replaces the evidence with the given id with the given evidence.
func (a *Area) EditEvidence(id int, evi string) {
	a.mu.Lock()
	if len(a.evidence) >= id {
		a.evidence[id] = evi
	}
	a.mu.Unlock()
}

func (a *Area) UpdateBuffer(s string) {
	a.mu.Lock()
	a.buffer = append(a.buffer[1:], s)
	a.mu.Unlock()
}

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

func (a *Area) CMs() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cms
}

func (a *Area) AddCM(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if sliceutil.ContainsInt(a.cms, uid) {
		return false
	}
	a.cms = append(a.cms, uid)
	return true
}

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

func (a *Area) HasCM(uid int) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return sliceutil.ContainsInt(a.cms, uid)
}

func (a *Area) EvidenceMode() EvidenceMode {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.evi_mode
}

func (a *Area) IniswapAllowed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Allow_iniswap
}

func (a *Area) NoInterrupt() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.Force_noint
}

func (a *Area) LastMsgID() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.last_msg
}

func (a *Area) SetLastMsgID(id int) {
	a.mu.Lock()
	a.last_msg = id
	a.mu.Unlock()
}
