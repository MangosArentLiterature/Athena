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
	"testing"
)

func TestUid(t *testing.T) {
	var uids UidManager
	uids.InitHeap(100)
	if len(uids.heap) != 100 {
		t.Errorf("unexpected heap length: got %d, want %d", len(uids.heap), 100)
	}
	uid1 := uids.GetUid()
	if uid1 != 0 {
		t.Errorf("uid1 = %d, want %d", uid1, 0)
	}
	uid2 := uids.GetUid()
	if uid2 != 1 {
		t.Errorf("uid2 = %d, want %d", uid2, 1)
	}
	uids.ReleaseUid(uid1)
	uid3 := uids.GetUid()
	if uid3 != 0 {
		t.Errorf("uid3 = %d, want %d", uid3, 0)
	}
	if len(uids.heap) != 98 {
		t.Errorf("unexpected heap length: got %d, want %d", len(uids.heap), 98)
	}
}
