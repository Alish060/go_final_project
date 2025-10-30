package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

var db *sql.DB

const schema = `
CREATE TABLE scheduler (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date CHAR(8) NOT NULL DEFAULT "",
	title VARCHAR(255) NOT NULL DEFAULT "",
	comment TEXT DEFAULT "",
	repeat VARCHAR(50) DEFAULT ""
);
CREATE INDEX idx_scheduler_date ON scheduler(date);
`

func Init(dbFile string) error {

	dbPath := os.Getenv("TODO_DBFILE")
	if dbPath == "" {
		dbPath = "./scheduler.db"
	}
	_, err := os.Stat(dbPath)
	fileExists := !os.IsNotExist(err)

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("ошибка при открытии базы: %w", err)
	}

	if !fileExists {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("ошибка при создании схемы: %w", err)
		}
		fmt.Println("База данных и таблица scheduler успешно созданы.")
	}

	return nil
}

func Close() {
	if err := db.Close(); err != nil {
		log.Printf("Ошибка закрытия БД: %v", err)
	}
}
