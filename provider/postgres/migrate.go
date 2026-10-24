package postgres

import (
	"ia-extraction/config"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (db postgresDB) MigrateDB() {
	dbOnly, _ := db.connection.DB()
	driver, err := postgres.WithInstance(dbOnly, &postgres.Config{})
	if err != nil {
		log.Println("could not start sql migration... ", err)
	} else {
		log.Println("start migration")

		m, err := migrate.NewWithDatabaseInstance(
			config.DbMigrationFile, config.DbName, driver)

		if err != nil {
			log.Println("migration create instance failed... ", err)
		}
		err = m.Up()
		if err != nil {
			log.Println("migration up failed... ", err)
		}

		log.Println("DatabaseImpl migrated")
	}
}

func (db postgresDB) DownDBMigration() {

	dbOnly, _ := db.connection.DB()
	driver, err := postgres.WithInstance(dbOnly, &postgres.Config{})

	if err != nil {
		log.Println("could not start sql migration... ", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		config.DbMigrationFile, config.DbName, driver)

	if err != nil {
		log.Println("migration failed... ", err)
	}
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		log.Println("An error occurred while removing the database.. ", err)
	}
	if err := m.Drop(); err != nil && err != migrate.ErrNoChange {
		log.Println("An error occurred while removing the database.. ", err)
	}

	log.Println("down Database migrated ")
}

func (db postgresDB) DownDBMigrationAndClose() {
	db.DownDBMigration()
	dbToClose, _ := db.connection.DB()
	dbToClose.Close()
}
