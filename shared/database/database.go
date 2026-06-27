package database

import (
	"fmt"
	"log"
	"time"

	"github.com/Yoak3n/gulu/util"
	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db *gorm.DB
)

func DB() *gorm.DB {
	return db
}

func initSqlite(name string) *gorm.DB {
	util.CreateDirNotExists("data/db")
	dsn := fmt.Sprintf("data/db/%s.db", name)
	sdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}
	return sdb
}

func InitDatabase() {
	switch config.Conf.Database.Type {
	case "postgres", "postgresql", "pgsql", "postgreSQL":
		db = initPostgres(config.Conf.Database.Host, config.Conf.Database.User, config.Conf.Database.Password, config.Conf.Database.Name, config.Conf.Database.Port)
	case "sqlite", "sqlite3":
		db = initSqlite(config.Conf.Database.Name)
	default:
		panic("Unsupported database type,please check the configuration")
	}
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(
		&table.UserTable{},
		&table.UserHistoryNameTable{},
		&table.VideoTable{},
		&table.LiveRoomTable{},
		&table.MedalTable{},
		&table.DanMuTable{},
		&table.CommentTable{},
		&table.SignedUserTable{},
		&table.UserEntryTable{},
		&table.ConfigurationTable{},
	)
	if err != nil {
		panic(err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)
	log.Println("database connected")
}
