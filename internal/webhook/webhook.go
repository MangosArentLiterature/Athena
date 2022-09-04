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

package webhook

import (
	"fmt"
	"strings"

	"github.com/ecnepsnai/discord"
)

var (
	ServerName  string
	ServerColor uint32 = 0x05b2f7
)

// PostModcall sends a modcall to the discord webhook.
func PostModcall(character string, area string, reason string) error {
	e := discord.Embed{
		Title:       fmt.Sprintf("%v sent a modcall in %v.", character, area),
		Description: reason,
		Color:       ServerColor,
	}
	p := discord.PostOptions{
		Username: ServerName,
		Embeds:   []discord.Embed{e},
	}
	err := discord.Post(p)
	return err
}

// PostReport sends a report file to the discord webhook.
func PostReport(name string, contents string) error {
	c := strings.NewReader(contents)
	f := discord.FileOptions{
		FileName: name,
		Reader:   c,
	}
	p := discord.PostOptions{
		Username: ServerName,
	}
	err := discord.UploadFile(p, f)
	return err
}
