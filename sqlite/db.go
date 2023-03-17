package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // Register relevant drivers.
)

func OpenDatabase(file string) (*sql.DB, error) {
	db, err := sql.Open(
		"sqlite",
		fmt.Sprintf(
			"file://%s?_pragma=foreign_keys=on&_pragma=journal_mode=WAL&_txlock=immediate",
			file,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	return db, nil
}
