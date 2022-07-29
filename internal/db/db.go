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

	_ "modernc.org/sqlite"
)

var DBPath string
var db *sql.DB

func Open() error {
	var err error
	db, err = sql.Open("sqlite", DBPath)
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS BANS(ID INT PRIMARY KEY, IPID TEXT, HDID TEXT, TIME INT, DURATION INT, REASON TEXT, MODERATOR TEXT)")
	if err != nil {
		return err
	}
	return nil
}

func Close() {
	db.Close()
}
