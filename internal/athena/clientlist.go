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

import "sync"

type ClientList struct {
	list map[*Client]struct{}
	mu   sync.Mutex
}

// AddClient adds a client to the list.
func (cl *ClientList) AddClient(c *Client) {
	cl.mu.Lock()
	cl.list[c] = struct{}{}
	cl.mu.Unlock()
}

// RemoveClient removes a client from the list.
func (cl *ClientList) RemoveClient(c *Client) {
	cl.mu.Lock()
	delete(cl.list, c)
	cl.mu.Unlock()
}

// GetAllClients returns all clients in the list.
func (cl *ClientList) GetAllClients() map[*Client]struct{} {
	return cl.list
}
