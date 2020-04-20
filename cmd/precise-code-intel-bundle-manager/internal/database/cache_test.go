package database

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDatabaseCacheEvictionWhileHeld(t *testing.T) {
	cache, err := NewDatabaseCache(2)
	if err != nil {
		t.Fatalf("unexpected error creating database cache: %s", err)
	}

	// reference to a db handle that outlives the cache entry
	var dbRef *Database

	// cache: foo
	if err := cache.WithDatabase("foo", openTestDatabase, func(db1 *Database) error {
		dbRef = db1

		// cache: bar,foo
		if err := cache.WithDatabase("bar", openTestDatabase, func(db *Database) error {
			return nil
		}); err != nil {
			return err
		}

		// cache: baz, bar
		// note: foo was evicted but should not be closed
		if err := cache.WithDatabase("baz", openTestDatabase, func(db *Database) error {
			return nil
		}); err != nil {
			return err
		}

		// cache: foo, bar
		// note: this version of foo should be a fresh connection
		return cache.WithDatabase("foo", openTestDatabase, func(db2 *Database) error {
			if db1 == db2 {
				return fmt.Errorf("unexpected cached database")
			}

			// evicted database stays open while held
			_ = readMetaLoop(db1)
			meta1, err1 := ReadMeta(db1.db)
			meta2, err2 := ReadMeta(db2.db)

			if err1 != nil {
				return err1
			}
			if err2 != nil {
				return err2
			}
			if meta1.LSIFVersion != "0.4.3" || meta2.LSIFVersion != "0.4.3" {
				return fmt.Errorf("unexpected lsif version: want=%s have=%s %s", "0.4.3", meta1.LSIFVersion, meta2.LSIFVersion)
			}

			return nil
		})
	}); err != nil {
		t.Fatalf("unexpected error during test: %s", err)
	}

	// evicted database is eventually closed
	if err := readMetaLoop(dbRef); err == nil {
		t.Fatalf("unexpected nil error")
	} else if !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("unexpected error: want=%s have=%s", "database is closed", err)
	}
}

func readMetaLoop(db *Database) (err error) {
	for i := 0; i < 100; i++ {
		if _, err = ReadMeta(db.db); err != nil {
			break
		}

		time.Sleep(time.Millisecond)
	}

	return
}
