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
	"strconv"
	"strings"
)

type cmdParamList struct {
	list *[]string
}

func (v cmdParamList) String() string {
	if v.list != nil {
		return strings.Join(*v.list, ",")
	}
	return ""
}

func (v cmdParamList) Set(s string) error {
	x := strings.Split(s, ",")
	*v.list = x
	return nil
}

// getUidList returns a list of clients that have the given UID(s).
func getUidList(uids []string) []*Client {
	var l []*Client
	for _, s := range uids {
		uid, err := strconv.Atoi(s)
		if err != nil || uid == -1 {
			continue
		}
		c, err := getClientByUid(uid)
		if err != nil {
			continue
		}
		l = append(l, c)
	}
	return l
}

// getIpidList returns a list of clients that have the given IPID(s).
func getIpidList(ipids []string) []*Client {
	var l []*Client
	for _, s := range ipids {
		c := getClientsByIpid(s)
		l = append(l, c...)
	}
	return l
}
