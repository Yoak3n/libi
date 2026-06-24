package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type (
	Configuration struct {
		RoomId    int       `yaml:"room_id"`
		Proxy     string    `yaml:"proxy"`
		Extension bool      `yaml:"extension"`
		Auth      *Auth     `yaml:"auth"`
		Database  *Database `yaml:"database"`
		CacheTTL  int       `yaml:"cache_ttl_hours"`
	}
	Auth struct {
		Cookie       string `yaml:"cookie"`
		RefreshToken string `yaml:"refresh_token"`
		ImgKey       string `yaml:"img_key"`
		SubKey       string `yaml:"sub_key"`
		LastUpdate   int64  `yaml:"last_update"`
	}
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"database"`
	}
)

var (
	Conf *Configuration
	v    *viper.Viper
)

func init() {
	_, err1 := os.Stat("config.yaml")
	_, err2 := os.Stat("config.yml")
	if os.IsNotExist(err1) && os.IsNotExist(err2) {
		fp, _ := os.Create("config.yaml")
		defer fp.Close()
	}
	v = viper.New()
	Conf = &Configuration{
		Auth:     &Auth{},
		Database: &Database{},
	}
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	err := v.ReadInConfig()
	if err != nil {
		return
	}
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.name", "bliveDB")
	v.SetDefault("ws_port", 10421)
	v.SetDefault("cache_ttl_hours", 24)
	getConfigFromFile()
	v.WatchConfig()
}

func getConfigFromFile() {
	_ = v.Unmarshal(Conf)
}

func SetWBIKey(img string, sub string) {
	v.Set("auth.img_key", img)
	v.Set("auth.sub_key", sub)
	Conf.Auth.ImgKey = img
	Conf.Auth.SubKey = sub
	t := time.Now().Unix()
	Conf.Auth.LastUpdate = t
	v.Set("auth.last_update", t)
	err := v.WriteConfig()
	if err != nil {
		return
	}
}

func SaveAuth() {
	v.Set("auth.cookie", Conf.Auth.Cookie)
	v.Set("auth.refresh_token", Conf.Auth.RefreshToken)
	_ = v.WriteConfig()
}

func SetRoomId(id int) error {
	v.Set("room_id", id)
	Conf.RoomId = id
	err := v.WriteConfig()
	if err != nil {
		return err
	}
	return nil
}
