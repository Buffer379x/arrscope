package app

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// Clone the legacy database using SQLite, including committed WAL pages. Keep
// the original intact for rollback; never rename live SQLite sidecar files.
func migrateLegacyDatabase(dir string) error {
	destination := filepath.Join(dir, "scopearr.db")
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	legacy := filepath.Join(dir, "arrscope.db")
	if _, err := os.Stat(legacy); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	db, err := sql.Open("sqlite", legacy+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	temporary := destination + ".migrating"
	// An interrupted clone is never used as the active database.
	if err = os.Remove(temporary); err != nil && !os.IsNotExist(err) {
		return err
	}
	if _, err = db.Exec("VACUUM INTO ?", temporary); err != nil {
		return fmt.Errorf("Could not migrate the legacy database: %w", err)
	}
	if err = os.Chmod(temporary, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}
