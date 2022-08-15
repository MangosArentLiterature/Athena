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

package permissions

import (
	"math"
)

type Role struct {
	Name        string   `toml:"name"`
	Permissions []string `toml:"permissions"`
}

var PermissionField = map[string]uint64{
	"NONE":        0,
	"CM":          1,
	"KICK":        1 << 1,
	"BAN":         1 << 2,
	"BYPASS_LOCK": 1 << 3,
	"ADMIN":       math.MaxInt64,
}

// GetPermissions returns the permissions for a role.
func (r *Role) GetPermissions() uint64 {
	var last uint64
	var current uint64
	for _, perm := range r.Permissions {
		current = PermissionField[perm] | last
		last = current
	}
	return current
}

// HasPermission checks if the supplied permissions matches the required permissions.
func HasPermission(perm uint64, required uint64) bool {
	return required == (perm & required)
}
