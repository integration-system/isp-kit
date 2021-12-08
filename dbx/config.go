package dbx

import (
	"fmt"
)

type Config struct {
	Host        string `valid:"required" schema:"Адрес"`
	Port        string `valid:"required" schema:"Порт"`
	Schema      string `valid:"required" schema:"Схема"`
	Database    string `valid:"required" schema:"Название базы данных"`
	Username    string `schema:"Логин"`
	Password    string `schema:"Пароль"`
	MaxOpenConn int    `schema:"Максимально возможное количество соединений,если <=0 - ограничений нет"`
}

func (c Config) Dsn() string {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.Database,
	)
	if c.Username != "" {
		dsn = fmt.Sprintf("%s user=%s", dsn, c.Username)
	}
	if c.Password != "" {
		dsn = fmt.Sprintf("%s password=%s", dsn, c.Password)
	}
	if c.Schema != "" {
		dsn = fmt.Sprintf("%s search_path=%s", dsn, c.Schema)
	}

	return dsn
}
