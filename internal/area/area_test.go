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

package area

import (
	"testing"
)

func TestJoin(t *testing.T) {
	a := NewArea(AreaData{}, 50, 0, EviAny)

	// Add a new client with CharID 0.
	// They should be the only one there.
	if !a.AddChar(0) {
		t.Errorf("adding new player to area: got %t, want %t", false, true)
	}
	if a.players != 1 {
		t.Errorf("unexpected value for area player count, got %d, want %d", a.players, 1)
	}

	// A second client with CharID 1 joins.
	// There are now two clients.
	if !a.AddChar(1) {
		t.Errorf("adding new player to area: got %t, want %t", false, true)
	}
	if a.players != 2 {
		t.Errorf("unexpected value for area player count, got %d, want %d", a.players, 2)
	}

	// A third client attempts to join with CharID 0.
	// Since a player has already taken that, it should fail.
	if a.AddChar(0) {
		t.Errorf("adding invalid player to area: got %t, want %t", true, false)
	}
	if a.players != 2 {
		t.Errorf("unexpected value for area player count, got %d, want %d", a.players, 2)
	}

	// The client with CharID 0 leaves.
	// Another client can now join with CharID 0.
	a.RemoveChar(0)
	if !a.AddChar(0) {
		t.Errorf("adding new player to area: got %t, want %t", false, true)
	}

	// A spectator joins the area.
	if !a.AddChar(-1) {
		t.Errorf("adding spectator to area: got %t, want %t", false, true)
	}
	if a.players != 3 {
		t.Errorf("unexpected value for area player count, got %d, want %d", a.players, 3)
	}
}

func TestSwitch(t *testing.T) {
	a := NewArea(AreaData{}, 50, 0, EviAny)

	// A client with CharID 0 joins, and switches to CharID 1.
	a.AddChar(0)
	if !a.SwitchChar(0, 1) {
		t.Errorf("switching empty character, got %t, want %t", false, true)
	}
	if a.AddChar(1) {
		t.Errorf("adding invalid player to area: got %t, want %t", true, false)
	}

	// A client with CharID 0 joins.
	if !a.AddChar(0) {
		t.Errorf("adding new player to area: got %t, want %t", false, true)
	}
	if a.SwitchChar(0, 1) {
		t.Errorf("switching taken character, got %t, want %t", true, false)
	}

	// A spectator joins
	a.AddChar(-1)
	if a.SwitchChar(-1, 0) {
		t.Errorf("switching taken character, got %t, want %t", true, false)
	}
}

func TestEvidence(t *testing.T) {
	a := NewArea(AreaData{}, 50, 0, EviAny)

	// Two pieces of evidence are added.
	evi1 := "foo&foo&foo"
	evi2 := "bar&bar&bar"
	a.AddEvidence(evi1)
	a.AddEvidence(evi2)
	if len(a.evidence) != 2 {
		t.Errorf("unexpected value for evidence length, got %d, want %d", len(a.evidence), 2)
	}

	// Evidence at indexes 0 and 1 are swapped.
	a.SwapEvidence(0, 1)
	if a.evidence[0] != evi2 {
		t.Errorf("unexpected value for evidence[0], got %s, want %s", a.evidence[0], evi2)
	}
	if a.evidence[1] != evi1 {
		t.Errorf("unexpected value for evidence[1], got %s, want %s", a.evidence[1], evi1)
	}

	// Evidence at index 0 is edited
	evi3 := "foobar&foobar&foobar"
	a.EditEvidence(0, evi3)
	if a.evidence[0] != evi3 {
		t.Errorf("unexpected value for evidence[0], got %s, want %s", a.evidence[0], evi3)
	}

	// Evidence at index 0 is removed.
	a.RemoveEvidence(0)
	if len(a.evidence) != 1 {
		t.Errorf("unexpected value for evidence length, got %d, want %d", len(a.evidence), 1)
	}
	if a.evidence[0] != evi1 {
		t.Errorf("unexpected value for evidence[0], got %s, want %s", a.evidence[0], evi1)
	}
}

func TestCMs(t *testing.T) {
	a := NewArea(AreaData{}, 50, 0, EviAny)

	// New CM is added to the area.
	if !a.AddCM(0) {
		t.Errorf("adding new cm to area: got %t, want %t", false, true)
	}
	if !a.HasCM(0) {
		t.Errorf("checking HasCM for joined cm: got %t, want %t", false, true)
	}

	// Adding CM with same UID should fail.
	if a.AddCM(0) {
		t.Errorf("adding invalid cm to area: got %t, want %t", true, false)
	}

	// CM is removed
	if !a.RemoveCM(0) {
		t.Errorf("removing cm from area: got %t, want %t", false, true)
	}
	if a.HasCM(0) {
		t.Errorf("checking HasCM for invalid cm: got %t, want %t", true, false)
	}
}

func TestInvited(t *testing.T) {
	a := NewArea(AreaData{}, 50, 0, EviAny)

	// New user is added to invite list
	if !a.AddInvited(1) {
		t.Errorf("adding new user to area invited: got %t, want %t", false, true)
	}
	if a.invited[0] != 1 {
		t.Errorf("unexpected value for invited[0], got %d, want %d", a.invited[0], 1)
	}

	// Adding invite with same UID should fail
	if a.AddInvited(1) {
		t.Errorf("adding invalid user to area invited: got %t, want %t", true, false)
	}

	// Remove invited user
	if !a.RemoveInvited(1) {
		t.Errorf("removing user to area invited: got %t, want %t", false, true)
	}
	if len(a.invited) != 0 {
		t.Errorf("unexpected value for invited length, got %d, want %d", len(a.invited), 0)
	}
}
