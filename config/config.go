package config

import (
	"github.com/jinzhu/configor"
)

type Config struct {
	AppConfig           AppConfig
	DBConfig            DBConfig
	AnimeScheduleConfig AnimeScheduleConfig
}

type AppConfig struct {
	APPName string `default:"animeschedule-sync"`
	Version string `default:"x.x.x" env:"VERSION"`
}

type DBConfig struct {
	Host     string `default:"localhost" env:"DBHOST"`
	DataBase string `default:"weeb" env:"DBNAME"`
	User     string `default:"weeb" env:"DBUSERNAME"`
	Password string `required:"true" env:"DBPASSWORD" default:"mysecretpassword"`
	Port     uint   `default:"3306" env:"DBPORT"`
	SSLMode  string `default:"false" env:"DBSSL"`
}

type AnimeScheduleConfig struct {
	BaseURL      string `default:"https://animeschedule.net/api/v3" env:"ANIMESCHEDULE_BASE_URL"`
	Token        string `default:"" env:"ANIMESCHEDULE_TOKEN"`
	SyncInterval string `default:"60m" env:"SYNC_INTERVAL"`
}

func LoadConfigOrPanic() Config {
	var config = Config{}
	configor.Load(&config, "config/config.dev.json")
	return config
}
