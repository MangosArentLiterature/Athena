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

import "sync"

type Area struct {
	AreaData
	mu       sync.Mutex
	taken    []bool
	players  int
	defhp    int
	prohp    int
	evidence []string
}

type AreaData struct {
	Name string `toml:"name"`
}

//Returns a new area
func NewArea(data AreaData, charlen int) *Area {
	return &Area{
		AreaData: data,
		taken:    make([]bool, charlen),
		defhp:    10,
		prohp:    10,
	}
}

// Returns the list of taken characters in an area, where "-1" is taken and "0" is free
func (a *Area) GetTaken() []string {
	a.mu.Lock()
	takenList := make([]string, len(a.taken))
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
func (a *Area) Join(char int) bool {
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
func (a *Area) Switch(old int, new int) bool {
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
func (a *Area) Leave(char int) {
	a.mu.Lock()
	if char != -1 {
		a.taken[char] = false
	}
	a.players--
	a.mu.Unlock()
}

// Returns the values of the def and pro HP bars.
func (a *Area) GetHP() (int, int) {
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
func (a *Area) GetPlayers() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.players
}

// Returns a list of evidence in the area.
func (a *Area) GetEvidence() []string {
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
