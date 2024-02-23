package main

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

type Operation struct {
	ID            uint   `gorm:"primaryKey"`
	Name          string `gorm:"not null"`
	ExecutionTime int    `gorm:"not null"`
}

type Expression struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4()"`
	Expression string         `gorm:"not null"`
	Status     string         `gorm:"default:'Pending'"`
	Tokens     pq.StringArray `gorm:"type:text[]"`
	Agent      *string
	Result     *float64

	CreatedAt   time.Time
	ProcessAt   *time.Time
	CompletedAt *time.Time
}

type Agent struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Active    bool
	LastSeen  time.Time
	CreatedAt time.Time
}

func initDatabase() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,         // Don't include params in the SQL log
		},
	)

	var err error
	databaseURL := os.Getenv("POSTGRES_URL")
	db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	err = db.AutoMigrate(&Expression{}, &Operation{}, &Agent{})
	if err != nil {
		log.Fatalf("Error migrating the database: %v", err)
	}

}
