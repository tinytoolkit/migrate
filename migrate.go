package migrate

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/jackc/pgx/v5"
	"golang.org/x/exp/slices"
)

// Migration represents a database migration with a version, description, up and down functions.
type Migration struct {
	Version     uint
	Description string
	Up          func(tx pgx.Tx) error
	Down        func(tx pgx.Tx) error
}

// Migrations is a slice of Migration.
type Migrations []Migration

// Sorted returns a sorted slice of migrations based on their versions.
func (ms *Migrations) Sorted() []Migration {
	sortedMigrations := make([]Migration, len(*ms))
	copy(sortedMigrations, *ms)

	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version < sortedMigrations[j].Version
	})

	return sortedMigrations
}

// Database represents a database connection and migration data.
type Database struct {
	conn           *pgx.Conn
	migrationTable string
	migrations     *Migrations
}

// New creates a new database instance with a DSN string and migrations.
func New(ctx context.Context, dsn string, migrations *Migrations) (*Database, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return NewWithConn(ctx, conn, migrations)
}

// NewWithConn creates a new database instance with a connection and migrations.
func NewWithConn(ctx context.Context, conn *pgx.Conn, migrations *Migrations) (*Database, error) {
	db := &Database{conn: conn, migrationTable: "migrations", migrations: migrations}
	return db, nil
}

// SetMigrationTable sets the name of the migration table.
func (db *Database) SetMigrationTable(table string) *Database {
	db.migrationTable = table
	return db
}

// MigrateUp migrates the database up to the current version (highest version).
func (db *Database) MigrateUp(ctx context.Context) error {
	tx, err := db.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = db.MigrationTable(ctx)
	if err != nil {
		return err
	}

	index, err := db.MigrationIndex(ctx)
	if err != nil {
		return err
	}

	for _, migration := range db.migrations.Sorted() {
		if migration.Version == 0 || migration.Description == "" {
			return fmt.Errorf("invalid migration: version and description must be set")
		}

		if migration.Up == nil || migration.Down == nil {
			return fmt.Errorf("invalid migration: up and down must be set")
		}

		if slices.Contains(index, migration.Version) {
			log.Printf("skipping migration (version=%v, description=%s) already exists", migration.Version, migration.Description)
			continue
		}

		if err := migration.Up(tx); err != nil {
			return err
		}

		if err := db.InsertMigration(ctx, migration.Version, migration.Description); err != nil {
			return err
		}

		log.Printf("migration up (version=%v, description=%s)", migration.Version, migration.Description)
	}
	return tx.Commit(ctx)
}

// MigrateDown migrates the database down by the specified amount.
func (db *Database) MigrateDown(ctx context.Context, amount int) error {
	tx, err := db.conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = db.MigrationTable(ctx)
	if err != nil {
		return err
	}

	index, err := db.MigrationIndex(ctx)
	if err != nil {
		return err
	}

	if len(index) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	if amount > len(index) {
		amount = len(index)
	}

	for i := len(index) - 1; i >= len(index)-amount; i-- {
		migration := db.migrations.Sorted()[index[i]-1]

		if migration.Version == 0 || migration.Description == "" {
			return fmt.Errorf("invalid migration: version and description must be set")
		}

		if migration.Up == nil || migration.Down == nil {
			return fmt.Errorf("invalid migration: up and down must be set")
		}

		if !slices.Contains(index, migration.Version) {
			return fmt.Errorf("migration (version=%v, description=%s) doesn't exists", migration.Version, migration.Description)
		}

		if err := migration.Down(tx); err != nil {
			return err
		}

		if err := db.DeleteMigration(ctx, migration.Version); err != nil {
			return err
		}

		log.Printf("migration down (version=%v, description=%s)", migration.Version, migration.Description)
	}
	return tx.Commit(ctx)
}

// MigrationTable creates the migration table if it doesn't exist.
func (db *Database) MigrationTable(ctx context.Context) error {
	_, err := db.conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			version INT UNIQUE NOT NULL,
			description VARCHAR(255) UNIQUE NOT NULL
		);
	`, db.migrationTable))
	return err
}

// MigrationIndex returns the migration index of the database.
func (db *Database) MigrationIndex(ctx context.Context) ([]uint, error) {
	query := fmt.Sprintf("SELECT version FROM %s ORDER BY version ASC;", db.migrationTable)

	rows, err := db.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var index []uint
	for rows.Next() {
		var version uint
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		index = append(index, version)
	}
	return index, nil
}

// InsertMigration inserts a migration into the migration table.
func (db *Database) InsertMigration(ctx context.Context, version uint, description string) error {
	query := fmt.Sprintf("INSERT INTO %s (version, description) VALUES ($1, $2);", db.migrationTable)
	_, err := db.conn.Exec(ctx, query, version, description)
	return err
}

// DeleteMigration deletes a migration from the migration table.
func (db *Database) DeleteMigration(ctx context.Context, version uint) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = $1;", db.migrationTable)
	_, err := db.conn.Exec(ctx, query, version)
	return err
}