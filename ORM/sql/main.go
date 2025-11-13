package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, _ := sql.Open("sqlite3", "orm.db")
	defer func() {
		_ = db.Close()
	}()

	_, _ = db.Exec("DROP TABLE IF EXISTS User;")
	_, _ = db.Exec("CREATE TABLE User(Name text);")
	ret, err := db.Exec("INSERT INTO User(`Name`) values (?), (?)", "Jw", "Boyue")
	if err == nil {
		affected, _ := ret.RowsAffected()
		log.Println(affected)
	}

	row := db.QueryRow("SELECT Name FROM User LIMIT 1;")
	var name string
	if err := row.Scan(&name); err == nil {
		log.Println(name)
	}

}
