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

package playercount

import "sync"

type PlayerCount struct {
	players int
	mu      sync.Mutex
}

// GetPlayerCount returns the current player count.
func (pc *PlayerCount) GetPlayerCount() int {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.players
}

// AddPlayer increments the player count by one.
func (pc *PlayerCount) AddPlayer() {
	pc.mu.Lock()
	pc.players++
	pc.mu.Unlock()
}

// RemovePlayer decrements the player count by one.
func (pc *PlayerCount) RemovePlayer() {
	pc.mu.Lock()
	pc.players--
	pc.mu.Unlock()
}
