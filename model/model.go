package model

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Model struct {
	DB *gorm.DB
}

func NewModel(dsn string, debug bool) (*Model, error) {
	var loglevel logger.LogLevel
	if debug {
		loglevel = logger.Info
	} else {
		loglevel = logger.Silent
	}
	var db *gorm.DB
	var err error
	db_config := strings.Split(dsn, "://")
	if len(db_config) != 2 {
		fmt.Println("invalid db uri")
		os.Exit(0)
	}
	db_type := db_config[0]
	db_uri := db_config[1]
	switch db_type {
	case "mysql":
		db, err = gorm.Open(mysql.Open(db_uri), &gorm.Config{
			Logger: logger.Default.LogMode(loglevel),
		})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(db_uri), &gorm.Config{
			Logger: logger.Default.LogMode(loglevel),
		})
	}
	if err != nil {
		return nil, err
	}
	sm := Model{
		DB: db,
	}
	db.AutoMigrate(&Data{})
	db.AutoMigrate(&ClientDeals{})
	db.AutoMigrate(&MarketDeal{})
	return &sm, nil
}
