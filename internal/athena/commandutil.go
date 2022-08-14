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

import "strconv"

func getKBList(usedList *[]string, useUid bool) []*Client {
	var l []*Client
	for _, s := range *usedList {
		if useUid {
			x, err := strconv.Atoi(s)
			if err != nil || x == -1 {
				continue
			}
			c, err := getClientByUid(x)
			if err != nil {
				continue
			}
			l = append(l, c)
		} else {
			c := getClientsByIpid(s)
			l = append(l, c...)
		}
	}
	return l
}
