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

// Package sliceutil provides functions for common slice operations.
package sliceutil

// ContainsString checks if a string is within a string slice.
func ContainsString(container []string, value string) bool {
	for _, x := range container {
		if x == value {
			return true
		}
	}
	return false
}

// ContainsString checks if an int is within an int slice.
func ContainsInt(container []int, value int) bool {
	for _, x := range container {
		if x == value {
			return true
		}
	}
	return false
}
