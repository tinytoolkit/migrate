package migrate

import "testing"

func TestMigrationSorting(t *testing.T) {
	migrations := Migrations{
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
