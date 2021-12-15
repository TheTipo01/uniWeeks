package main

import (
	"github.com/bwmarrin/lit"
)

const (
	tblUsers = "CREATE TABLE IF NOT EXISTS `users` ( `id` bigint(20) unsigned NOT NULL, `even` tinyint(1) NOT NULL, PRIMARY KEY (`id`) );"
)

// Executes a simple query
func execQuery(query string) {
	_, err := db.Exec(query)
	if err != nil {
		lit.Error("Error executing query: " + err.Error())
	}
}

func updateDB(id int64, even bool) {
	if val, ok := cache[id]; !ok {
		// The user is new, add line
		_, err := db.Exec("INSERT INTO users(id, even) VALUES (?, ?)", id, even)
		if err != nil {
			lit.Error("Error adding user: " + err.Error())
		}

		cache[id] = even
	} else {
		// Else if option are different, update db
		if even != val {
			_, err := db.Exec("UPDATE users SET even = ? WHERE id = ?", even, id)
			if err != nil {
				lit.Error("Error updating user: " + err.Error())
			}

			cache[id] = even
		}
	}
}

// Deletes the user from the database
func deleteFromDB(id int64) {
	_, _ = db.Exec("DELETE FROM users WHERE id = ?", id)
	delete(cache, id)
}

func loadCache() {
	var (
		id       int64
		userEven bool
	)
	rows, err := db.Query("SELECT id, even FROM users")
	if err != nil {
		lit.Error("Error while querying db: " + err.Error())
		return
	}

	// Iterate every user
	for rows.Next() {
		_ = rows.Scan(&id, &userEven)
		cache[id] = userEven
	}
}
