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

// Package packet implements AO2 network packets.
package packet

import (
	"fmt"
	"strings"
)

// Packet represents an AO2 network packet.
// AO2 network packets are comprised of a non-empty header, followed by a '#'-separated list of parameters, ending with a '%'.
type Packet struct {
	Header string
	Body   []string
}

// NewPacket returns a new Packet with the specified data, which should be a valid AO2 packet.
func NewPacket(data string) (*Packet, error) {
	p := &Packet{}
	s := strings.Split(data, "#")
	if strings.TrimSpace(s[0]) == "" {
		return nil, fmt.Errorf("packet header cannot be empty")
	}
	p.Header = s[0]
	s = s[1:] // Remove header
	if len(s) > 1 {
		s = s[:len(s)-1] // Remove empty entry after the final '#'
	}
	p.Body = s
	return p, nil
}

// String returns a string representation of the Packet.
func (p Packet) String() string {
	return p.Header + "#" + strings.Join(p.Body, "#") + "#%"
}
