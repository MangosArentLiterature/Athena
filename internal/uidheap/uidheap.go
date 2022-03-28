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

// Package uidheap implements a user ID heap
package uidheap

type UidHeap []int

func (u *UidHeap) Push(x interface{}) {
	*u = append(*u, x.(int))
}
func (u *UidHeap) Pop() interface{} {
	o := *u
	n := len(o)
	x := o[n-1]
	*u = o[0 : n-1]
	return x
}

func (u UidHeap) Less(i, j int) bool {
	return u[i] < u[j]
}

func (u UidHeap) Len() int {
	return len(u)
}

func (u UidHeap) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}
