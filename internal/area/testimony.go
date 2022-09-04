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
	"fmt"
	"strings"
)

// TstState returns the testimony recorder's current state.
func (a *Area) TstState() TRState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tr.State
}

// SetTstState sets the testimony recorder's state.
func (a *Area) SetTstState(s TRState) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tr.State = s
}

// CurrentTstStatement returns the testimony recorder's current statement.
func (a *Area) CurrentTstStatement() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tr.Testimony[a.tr.Index]
}

// CurrentTstIndex returns the testimony recorder's current index.
func (a *Area) CurrentTstIndex() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.tr.Index
}

// TstInsert inserts a new statement into the testimony.
func (a *Area) TstInsert(s string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tr.Index != 0 {
		x := strings.Split(s, "#")
		x[14] = "1"
		s = strings.Join(x, "#")
	}
	a.tr.Testimony = append(a.tr.Testimony, "")
	copy(a.tr.Testimony[a.tr.Index+2:], a.tr.Testimony[a.tr.Index+1:])
	a.tr.Testimony[a.tr.Index+1] = s
	return nil
}

// TstRemove removes the current statement from the testimony.
func (a *Area) TstRemove() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.tr.Testimony) < 2 {
		return fmt.Errorf("empty testimony")
	}
	a.tr.Testimony = append(a.tr.Testimony[:a.tr.Index], a.tr.Testimony[a.tr.Index+1:]...)
	return nil
}

// TstUpdate updates the testimony's current statement.
func (a *Area) TstUpdate(s string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tr.Index != 0 {
		x := strings.Split(s, "#")
		x[14] = "1"
		s = strings.Join(x, "#")
	}
	a.tr.Testimony[a.tr.Index] = s
	return nil
}

// TstAdvance advances the testimony forward by one statement.
func (a *Area) TstAdvance() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tr.Index == len(a.tr.Testimony)-1 {
		a.tr.Index = 1
	} else {
		a.tr.Index++
	}
}

// TstRewind advances the testimony backward by one statement.
func (a *Area) TstRewind() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tr.Index <= 1 {
		a.tr.Index = 1
	} else {
		a.tr.Index--
	}
}

// TstAppend appends a new statement to the testimony.
func (a *Area) TstAppend(s string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tr.Index != 0 {
		x := strings.Split(s, "#")
		x[14] = "1"
		s = strings.Join(x, "#")
	}
	a.tr.Testimony = append(a.tr.Testimony, s)
}

// TstClear clears the currently recorded testimony.
func (a *Area) TstClear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tr.Testimony = []string{}
	a.tr.Index = 0
}

// TstLen returns the length of the testimony.
func (a *Area) TstLen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.tr.Testimony)
}

// TstJump advanced the testimony to the given index.
func (a *Area) TstJump(i int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tr.Index = i
}
