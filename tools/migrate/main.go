package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/Yoak3n/libi/shared/domain/model/table"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func main() {
	sqlitePath := flag.String("sqlite", "", "SQLite database file path")
	pgDSN := flag.String("pg", "", "PostgreSQL DSN (e.g. \"host=localhost user=postgres password=xxx dbname=bliveDB port=5432 sslmode=disable TimeZone=Asia/Shanghai\")")
	batchSize := flag.Int("batch", 500, "Batch insert size")
	clean := flag.Bool("clean", false, "Truncate all PostgreSQL tables")
	flag.Parse()

	if *pgDSN == "" {
		log.Fatal("PostgreSQL DSN is required. Use -pg flag.")
	}

	// Open SQLite
	src, err := gorm.Open(sqlite.Open(*sqlitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to open SQLite: %v", err)
	}

	// Open PostgreSQL
	dst, err := gorm.Open(postgres.Open(*pgDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Auto-migrate schema on PostgreSQL
	if err := dst.AutoMigrate(
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
	); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	// Optionally truncate target tables
	if *clean {
		dst.Exec("SET session_replication_role = 'replica'")
		tables, _ := dst.Migrator().GetTables()
		for _, t := range tables {
			dst.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", t))
		}
		dst.Exec("SET session_replication_role = 'origin'")
		fmt.Printf("Truncated %d tables.\n", len(tables))
	}

	// Clean mode: truncate and exit
	if *clean && *sqlitePath == "" {
		fmt.Println("Clean only mode.")
		return
	}

	if *sqlitePath == "" {
		log.Fatal("SQLite path is required. Use -sqlite flag, or use --clean to only truncate PostgreSQL tables.")
	}

	// Disable foreign key checks for migration
	if err := dst.Exec("SET session_replication_role = 'replica'").Error; err != nil {
		log.Fatalf("Failed to disable FK checks: %v", err)
	}
	defer dst.Exec("SET session_replication_role = 'origin'")

	// Migrate tables in dependency order
	migrations := []struct {
		name  string
		model interface{}
	}{
		{"users", &[]table.UserTable{}},
		{"user_history_names", &[]table.UserHistoryNameTable{}},
		{"live_rooms", &[]table.LiveRoomTable{}},
		{"videos", &[]table.VideoTable{}},
		{"medals", &[]table.MedalTable{}},
		{"dan_mus", &[]table.DanMuTable{}},
		{"comments", &[]table.CommentTable{}},
		{"signed_users", &[]table.SignedUserTable{}},
		{"user_entries", &[]table.UserEntryTable{}},
		{"configurations", &[]table.ConfigurationTable{}},
	}

	for _, m := range migrations {
		fmt.Printf("Migrating %-20s ... ", m.name)
		count, err := migrateTable(src, dst, m.model, *batchSize)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Printf("%d rows\n", count)
		}
	}

	fmt.Println("Done.")
}

func migrateTable(src, dst *gorm.DB, model interface{}, batchSize int) (int, error) {
	var total int64
	if err := src.Model(model).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}
	if total == 0 {
		return 0, nil
	}

	if err := src.Find(model).Error; err != nil {
		return 0, fmt.Errorf("find: %w", err)
	}

	// Sanitize null bytes from all string fields
	sanitize(model)

	// Insert with ON CONFLICT DO NOTHING to skip duplicates
	if err := dst.Clauses(clause.OnConflict{DoNothing: true}).
		Session(&gorm.Session{CreateBatchSize: batchSize}).
		Create(model).Error; err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}

	return int(total), nil
}

// sanitize removes null bytes from string fields in the slice.
func sanitize(model interface{}) {
	switch rows := model.(type) {
	case *[]table.UserTable:
		for i := range *rows {
			(*rows)[i].Name = cleanStr((*rows)[i].Name)
			(*rows)[i].Avatar = cleanStr((*rows)[i].Avatar)
		}
	case *[]table.UserHistoryNameTable:
		for i := range *rows {
			(*rows)[i].Name = cleanStr((*rows)[i].Name)
		}
	case *[]table.VideoTable:
		for i := range *rows {
			(*rows)[i].Title = cleanStr((*rows)[i].Title)
			(*rows)[i].Cover = cleanStr((*rows)[i].Cover)
			(*rows)[i].Topic = cleanStr((*rows)[i].Topic)
			(*rows)[i].Description = cleanStr((*rows)[i].Description)
			(*rows)[i].Bvid = cleanStr((*rows)[i].Bvid)
		}
	case *[]table.LiveRoomTable:
		for i := range *rows {
			(*rows)[i].Title = cleanStr((*rows)[i].Title)
			(*rows)[i].Cover = cleanStr((*rows)[i].Cover)
		}
	case *[]table.MedalTable:
		for i := range *rows {
			(*rows)[i].Name = cleanStr((*rows)[i].Name)
		}
	case *[]table.DanMuTable:
		for i := range *rows {
			(*rows)[i].Content = cleanStr((*rows)[i].Content)
			(*rows)[i].MessageId = cleanStr((*rows)[i].MessageId)
		}
	case *[]table.CommentTable:
		for i := range *rows {
			(*rows)[i].Text = cleanStr((*rows)[i].Text)
		}
	case *[]table.SignedUserTable:
		for i := range *rows {
			(*rows)[i].Description = cleanStr((*rows)[i].Description)
		}
	case *[]table.ConfigurationTable:
		for i := range *rows {
			(*rows)[i].Data = cleanStr((*rows)[i].Data)
			(*rows)[i].RefreshToken = cleanStr((*rows)[i].RefreshToken)
			(*rows)[i].Type = cleanStr((*rows)[i].Type)
		}
	}
}

func cleanStr(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}
