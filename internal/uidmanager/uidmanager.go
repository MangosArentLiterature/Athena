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

package uidmanager

import (
	"container/heap"
	"sync"

	"github.com/MangosArentLiterature/Athena/internal/uidheap"
)

type UidManager struct {
	heap uidheap.UidHeap
	mu   sync.Mutex
}

func (u *UidManager) InitHeap(players int) {
	u.mu.Lock()
	u.heap = make(uidheap.UidHeap, players)
	for i := range u.heap {
		u.heap[i] = i
	}
	heap.Init(&u.heap)
	u.mu.Unlock()
}

func (u *UidManager) GetUid() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return heap.Pop(&u.heap).(int)
}

func (u *UidManager) ReleaseUid(uid int) {
	u.mu.Lock()
	heap.Push(&u.heap, uid)
	u.mu.Unlock()
}
