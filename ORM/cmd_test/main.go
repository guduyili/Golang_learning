package main

import (
	"fmt"
	"log"
	"orm"

	_ "modernc.org/sqlite"
)

func main() {
	// 使用 modernc.org/sqlite 纯 Go 驱动，驱动名为 "sqlite"
	engine, err := orm.NewEngine("sqlite", "orm.db")
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	defer engine.Close()

	s := engine.NewSession()

	if _, err := s.Raw("DROP TABLE IF EXISTS User;").Exec(); err != nil {
		log.Fatalf("drop table failed: %v", err)
	}
	// if _, err := s.Raw("CREATE TABLE IF NOT EXISTS User(Name TEXT);").Exec(); err != nil {
	// 	log.Fatalf("create table failed: %v", err)
	// }
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()
	_, _ = s.Raw("CREATE TABLE User(Name text);").Exec()

	ret, err := s.Raw("INSERT INTO User(`Name`) VALUES (?),(?);", "Jw", "Boyue").Exec()
	if err != nil {
		log.Fatalf("insert failed: %v", err)
	}
	count, err := ret.RowsAffected()
	if err != nil {
		log.Fatalf("rows affected failed: %v", err)
	}
	fmt.Printf("Exec success, %d affected\n", count)
}
