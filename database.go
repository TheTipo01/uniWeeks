package main

import "github.com/bwmarrin/lit"

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

func updateDB(id int, even bool) {
	var savedEven bool

	err := db.QueryRow("SELECT even FROM users WHERE id = ?", id).Scan(&savedEven)
	// The user is new, add line
	if err != nil {
		_, err = db.Exec("INSERT INTO users(id, even) VALUES (?, ?)", id, even)
		if err != nil {
			lit.Error("Error adding user: " + err.Error())
		}
		return
	}

	// Else if option are different, update db
	if even != savedEven {
		_, err = db.Exec("UPDATE users SET even = ? WHERE id = ?", even, id)
		if err != nil {
			lit.Error("Error updating user: " + err.Error())
		}
		return
	}
}

// Deletes the user from the database
func deleteFromDB(id int) {
	_, _ = db.Exec("DELETE FROM users WHERE id = ?", id)
}
