package postgres

import (
	"fmt"
	"ia-extraction/config"
	"log"

	"github.com/lib/pq"
	sqltrace "github.com/signalfx/signalfx-go-tracing/contrib/database/sql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB interface {
	Connect() *gorm.DB
	MigrateDB()
	DownDBMigration()
	DownDBMigrationAndClose()
	GetConnection() *gorm.DB
	CloseConnection() error
}

func NewPostgresDB() PostgresDB {
	postgresDb := postgresDB{}
	log.Println("Init db")
	postgresDb.connection = postgresDb.Connect()
	postgresDb.MigrateDB()

	return postgresDb
}

type postgresDB struct {
	connection *gorm.DB
}

func (db postgresDB) Connect() *gorm.DB {
	sqltrace.Register("postgres", &pq.Driver{}, sqltrace.WithServiceName(config.MsName))
	connStr := db.connectionString()
	dbInstance, _ := gorm.Open(postgres.Open(connStr))
	return dbInstance
}

func (db postgresDB) connectionString() string {
	str := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.DbUser,
		config.DbPass,
		config.DbHost,
		config.DbPort,
		config.DbName,
		config.DbSSLMode,
	)

	return str
}

func (db postgresDB) GetConnection() *gorm.DB {
	if !config.ProdEnvironment {
		return db.connection.Debug()
	}
	return db.connection
}

func (db postgresDB) CloseConnection() error {
	dbToClose, _ := db.connection.DB()
	return dbToClose.Close()
}
