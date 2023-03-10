# tinytoolkit/migrate

A simple SQLite migration library for Go.

## Installation

```bash
go get github.com/tinytoolkit/migrate
```

## Example

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/tinytoolkit/migrate"

	_ "github.com/mattn/go-sqlite3"
)

func main() {

	// Create the migrations slice.
	migrations := migrate.Migrations{
		{
			Version:     1,
			Description: "create users table",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`
					CREATE TABLE IF NOT EXISTS users (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						name TEXT NOT NULL
					);
				`)
				return err
			},
			Down: func(tx *sql.Tx) error {
				_, err := tx.Exec("DROP TABLE IF EXISTS users;")
				return err
			},
		},
		{
			Version:     2,
			Description: "add email column to users table",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec("ALTER TABLE users ADD COLUMN email TEXT;")
				return err
			},
			Down: func(tx *sql.Tx) error {
				_, err := tx.Exec("ALTER TABLE users DROP COLUMN email;")
				return err
			},
		},
	}

	// Create a new database instance.
	database, err := migrate.New("sqlite3", "test.sqlite?cache=shared&mode=rwc", &migrations)
	if err != nil {
		log.Fatalf("failed to create database instance: %v", err)
	}

	// Migrate up to the latest version.
	err = database.MigrateUp(context.Background())
	if err != nil {
		log.Fatalf("failed to migrate up: %v", err)
	}

	// Migrate down to the previous version.
	err = database.MigrateDown(context.Background(), 1)
	if err != nil {
		log.Fatalf("failed to migrate down: %v", err)
	}

	// Get the current version of the database.
	version, err := database.CurrentVersion(context.Background())
	if err != nil {
		log.Fatalf("failed to get current version: %v", err)
	}
	fmt.Printf("current database version: %d\n", version)
}
```
