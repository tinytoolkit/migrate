package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	"golang.org/x/exp/slices"

	_ "github.com/mattn/go-sqlite3"
)

// Migration represents a database migration with a version, description, up and down functions.
type Migration struct {
	Version     uint
	Description string
	Up          func(tx *sql.Tx) error
	Down        func(tx *sql.Tx) error
}

// Migrations is a slice of Migration.
type Migrations []Migration

// Sorted returns a sorted slice of migrations based on their versions.
func (ms *Migrations) sorted() []Migration {
	sortedMigrations := make([]Migration, len(*ms))
	copy(sortedMigrations, *ms)

	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version < sortedMigrations[j].Version
	})
	return sortedMigrations
}

// Database represents a database connection and migration data.
type Database struct {
	conn           *sql.DB
	migrationTable string
	migrations     *Migrations
}

// New creates a new database instance with a DSN string and migrations.
func New(dsn string, migrations *Migrations) (*Database, error) {
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return &Database{
		conn:           conn,
		migrationTable: "migrations",
		migrations:     migrations,
	}, nil
}

// NewWithConn creates a new database instance with a database connection and migrations.
func NewWithConn(conn *sql.DB, migrations *Migrations) *Database {
	return &Database{
		conn:           conn,
		migrationTable: "migrations",
		migrations:     migrations,
	}
}

// Close closes the database connection.
func (db *Database) Close() error {
	return db.conn.Close()
}

// SetMigrationTable sets the name of the migration table.
func (db *Database) SetMigrationTable(table string) *Database {
	db.migrationTable = table
	return db
}

// MigrateUp migrates the database up to the current version (highest version).
func (db *Database) MigrateUp(ctx context.Context) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = db.createMigrationTable(ctx, tx)
	if err != nil {
		return err
	}

	index, err := db.getMigrationIndex(ctx, tx)
	if err != nil {
		return err
	}

	for _, migration := range db.migrations.sorted() {
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

		if err := db.insertMigration(ctx, tx, migration.Version, migration.Description); err != nil {
			return err
		}

		log.Printf("migration up (version=%v, description=%s)", migration.Version, migration.Description)
	}
	return tx.Commit()
}

// MigrateDown migrates the database down by the specified amount.
func (db *Database) MigrateDown(ctx context.Context, amount int) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = db.createMigrationTable(ctx, tx)
	if err != nil {
		return err
	}

	index, err := db.getMigrationIndex(ctx, tx)
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
		migration := db.migrations.sorted()[index[i]-1]

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

		if err := db.deleteMigration(ctx, tx, migration.Version); err != nil {
			return err
		}

		log.Printf("migration down (version=%v, description=%s)", migration.Version, migration.Description)
	}
	return tx.Commit()
}

// CurrentVersion returns the current version of the database.
func (db *Database) CurrentVersion(ctx context.Context) (uint, error) {
	query := fmt.Sprintf("SELECT version FROM %s ORDER BY version DESC LIMIT 1;", db.migrationTable)

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}

	if !rows.Next() {
		return 0, nil
	}

	version := uint(0)
	if err := rows.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func (db *Database) createMigrationTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER UNIQUE NOT NULL,
			description VARCHAR(255) UNIQUE NOT NULL
		);
	`, db.migrationTable))
	return err
}

func (db *Database) getMigrationIndex(ctx context.Context, tx *sql.Tx) ([]uint, error) {
	query := fmt.Sprintf("SELECT version FROM %s ORDER BY version ASC;", db.migrationTable)

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := make([]uint, 0)
	for rows.Next() {
		var version uint
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		index = append(index, version)
	}
	return index, nil
}

func (db *Database) insertMigration(ctx context.Context, tx *sql.Tx, version uint, description string) error {
	query := fmt.Sprintf("INSERT INTO %s (version, description) VALUES (?, ?);", db.migrationTable)
	_, err := tx.ExecContext(ctx, query, version, description)
	return err
}

func (db *Database) deleteMigration(ctx context.Context, tx *sql.Tx, version uint) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE version = ?;", db.migrationTable)
	_, err := tx.ExecContext(ctx, query, version)
	return err
}
