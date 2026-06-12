package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"sso.pelajarnumagetan.or.id/internal/config"
)

var DB *gorm.DB

func ConnectPostgres() *gorm.DB {
	cfg := config.Get()

	var dsn string
	if cfg.DBPassword != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%s/%s?sslmode=disable&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = db
	return db
}
