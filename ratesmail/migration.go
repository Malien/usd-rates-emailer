package ratesmail

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var migrations = []string{
	`
    create table subscribers (
        id integer primary key autoincrement,
        email text not null unique
    );
    `,
}

func Migrate(db *sql.DB) error {
	_, err := db.Exec(`
        create table if not exists migrations (
            migration_order integer not null,
            applied_at integer not null
        );
    `)
	if err != nil {
		return err
	}

	var maxOrder *int
	err = db.QueryRow("select max(migration_order) from migrations").Scan(&maxOrder)
	if err != nil {
		return err
	}
	if maxOrder == nil {
		defaultMaxOrder := 0
		maxOrder = &defaultMaxOrder
	}

	relevantMigrations := migrations[*maxOrder:]
	for i, migration := range relevantMigrations {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		_, err = tx.Exec(migration)
		if err != nil {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("insert into migrations (migration_order, applied_at) values (?, unixepoch())", *maxOrder+i+1)
		if err != nil {
			tx.Rollback()
			return err
		}
		err = tx.Commit()
		if err != nil {
			return err
		}
		log.Printf("Migration %d applied", *maxOrder+i)
	}

	return nil
}

func OpenDB(conf DBConfig) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", conf.Filename)
	if err != nil {
		return nil, err
	}
	if conf.WALMode {
		_, err = db.Exec("pragma journal_mode = wal;")
		if err != nil {
			return nil, err
		}
	}
	err = Migrate(db)
	if err != nil {
		return nil, err
	}
	return db, nil
}
