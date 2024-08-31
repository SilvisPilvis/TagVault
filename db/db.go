package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var Db *sql.DB = nil

func init() {
	Db, err := sql.Open("sqlite3", "file:../index.db")
	if err != nil {
		panic(err.Error())
	}
	Db.SetMaxOpenConns(1)
	defer Db.Close()

	// check the connection
	err = Db.Ping()
	if err != nil {
		fmt.Print("Not Connected to db!\n")
		log.Fatal(err.Error(), "\n")
	}
	fmt.Print("Connected to db!\n")
	// return Db
}
