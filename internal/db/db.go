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

package db

import (
	"database/sql"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type BanInfo struct {
	Id        int
	Ipid      string
	Hdid      string
	Time      int64
	Duration  int64
	Reason    string
	Moderator string
}

type BanLookup int

const (
	IPID BanLookup = iota
	HDID
)

var DBPath string
var db *sql.DB

// Opens the server's database connection.
func Open() error {
	var err error
	db, err = sql.Open("sqlite", DBPath)
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BANS(ID INTEGER PRIMARY KEY, IPID TEXT, HDID TEXT, TIME INTEGER, DURATION INTEGER, REASON TEXT, MODERATOR TEXT)")
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS USERS(USERNAME TEXT PRIMARY KEY, PASSWORD TEXT, PERMISSIONS TEXT)")
	if err != nil {
		return err
	}
	return nil
}

// UserExists returns whether a user exists within the server's database.
func UserExists(username string) bool {
	result := db.QueryRow("SELECT USERNAME FROM USERS WHERE USERNAME = ?", username)
	if result.Scan() == sql.ErrNoRows {
		return false
	} else {
		return true
	}
}

// CreateUser adds a new user to the server's database.
func CreateUser(username string, password []byte, permissions uint64) error {
	hashed, err := bcrypt.GenerateFromPassword(password, 12)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO USERS VALUES(?, ?, ?)", username, hashed, strconv.FormatUint(permissions, 10))
	if err != nil {
		return err
	}
	return nil
}

// RemoveUser deletes a user from the server's database.
func RemoveUser(username string) error {
	_, err := db.Exec("DELETE FROM USERS WHERE USERNAME = ?", username)
	if err != nil {
		return err
	}
	return nil
}

// AuthenticateUser returns whether or not the user's credentials match those in the database, and that user's permissions.
func AuthenticateUser(username string, password []byte) (bool, uint64) {
	var rpass, rperms string
	result := db.QueryRow("SELECT PASSWORD, PERMISSIONS FROM USERS WHERE USERNAME = ?", username)
	result.Scan(&rpass, &rperms)
	err := bcrypt.CompareHashAndPassword([]byte(rpass), password)
	if err != nil {
		return false, 0
	}
	p, err := strconv.ParseUint(rperms, 10, 64)
	if err != nil {
		return false, 0
	}
	return true, p
}

func ChangePermissions(username string, permissions uint64) error {
	_, err := db.Exec("UPDATE USERS SET PERMISSIONS = ? WHERE USERNAME = ?", strconv.FormatUint(permissions, 10), username)
	if err != nil {
		return err
	}
	return nil
}

func AddBan(ipid string, hdid string, time int64, duration int64, reason string, moderator string) (int, error) {
	result, err := db.Exec("INSERT INTO BANS VALUES(NULL, ?, ?, ?, ?, ?, ?)", ipid, hdid, time, duration, reason, moderator)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func UnBan(id int) error {
	_, err := db.Exec("UPDATE BANS SET DURATION = 0 WHERE ID = ?", id)
	if err != nil {
		return err
	}
	return nil
}

func GetBan(id int) (BanInfo, error) {
	result := db.QueryRow("SELECT * FROM BANS WHERE ID = ?", id)
	var ban BanInfo
	if result.Scan(&ban.Id, &ban.Ipid, &ban.Hdid, &ban.Time, &ban.Duration, &ban.Reason, &ban.Moderator) == sql.ErrNoRows {
		return ban, sql.ErrNoRows
	} else {
		return ban, nil
	}
}

func IsBanned(by BanLookup, value string) (bool, BanInfo, error) {
	var stmt *sql.Stmt
	var err error
	switch by {
	case IPID:
		stmt, err = db.Prepare("SELECT ID, DURATION, REASON FROM BANS WHERE IPID = ?")
	case HDID:
		stmt, err = db.Prepare("SELECT ID, DURATION, REASON FROM BANS WHERE HDID = ?")
	}
	if err != nil {
		return false, BanInfo{}, err
	}
	result, err := stmt.Query(value)
	if err != nil {
		return false, BanInfo{}, err
	}
	stmt.Close()
	defer result.Close()
	for result.Next() {
		var duration int64
		var id int
		var reason string
		result.Scan(&id, &duration, &reason)
		if duration == -1 || time.Unix(duration, 0).UTC().After(time.Now().UTC()) {
			return true, BanInfo{Id: id, Duration: duration, Reason: reason}, nil
		}
	}
	return false, BanInfo{}, nil
}

func UpdateBan(id int, duration int, reason string) error {
	_, err := db.Exec("UPDATE BANS SET DURATION = ?, REASON = ? WHERE ID = ?", duration, reason, id)
	if err != nil {
		return err
	}
	return nil
}

// Closes the server's database connection.
func Close() {
	db.Close()
}
