# tinytoolkit/migrate

## Installation

```bash
go get github.com/tinytoolkit/migrate
```

## Example

```go
package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"

	"github.com/tinytoolkit/migrate"
)

func main() {
	ctx := context.Background()

	migrations := &migrate.Migrations{
		{
			Version:     1,
			Description: "create users table",
			Up: func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					CREATE TABLE users (
						id SERIAL PRIMARY KEY,
						name TEXT NOT NULL
					)
				`)
				return err
			},
			Down: func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					DROP TABLE users
				`)
				return err
			},
		},
		{
			Version:     2,
			Description: "create posts table",
			Up: func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					CREATE TABLE posts (
						id SERIAL PRIMARY KEY,
						user_id INT NOT NULL,
						title TEXT NOT NULL,
						body TEXT NOT NULL,
						FOREIGN KEY (user_id) REFERENCES users(id)
					)
				`)
				return err
			},
			Down: func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `
					DROP TABLE posts
				`)
				return err
			},
		},
	}

	dsn := "postgres://postgres:postgres@localhost:5432/postgres"

	db, err := migrate.New(ctx, dsn, migrations)
	if err != nil {
		log.Fatalln(err)
	}

	if err := db.MigrateUp(ctx); err != nil {
		log.Fatalln(err)
	}
}
```