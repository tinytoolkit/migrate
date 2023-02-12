package migrate_test

import (
	"testing"

	"github.com/tinytoolkit/migrate"
)

func TestMigrationSorting(t *testing.T) {
	migrations := migrate.Migrations{
		{Version: 1, Description: "first migration"},
		{Version: 4, Description: "fourth migration"},
		{Version: 3, Description: "third migration"},
		{Version: 5, Description: "fifth migration"},
		{Version: 2, Description: "second migration"},
	}

	sorted := migrations.Sorted()

	for i, m := range sorted {
		if i+1 != int(m.Version) {
			t.Errorf("expected version %d, got %d", i+1, m.Version)
		}
	}
}
