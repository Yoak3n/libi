// Package store 提供对 shared 数据库的只读访问。
// mirror 不创建自己的表，分析结果全走 JSON 文件。
package store

import (
	"log"

	sharedconfig "github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/database"
	"gorm.io/gorm"
)

var db *gorm.DB

// DB 返回 shared 的数据库连接（troll 的数据在这里）
func DB() *gorm.DB {
	return db
}

// Init 初始化 shared 数据库连接。
// 仅当当前目录有 config.yaml（troll 的配置）时才会成功。
// 没有配置时 db 为 nil，pipeline 会报错提示。
func Init() {
	// shared config 的 init() 在包导入时自动运行
	if sharedconfig.Conf == nil || sharedconfig.Conf.Database == nil || sharedconfig.Conf.Database.Type == "" {
		log.Println("[store] no shared config found (missing config.yaml?) — DB unavailable")
		return
	}

	database.InitDatabase()
	db = database.DB()
	if db != nil {
		log.Println("[store] shared database connected")
	}
}

// IsReady 返回数据库是否可用
func IsReady() bool {
	return db != nil
}
