package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Env      string   `envconfig:"APP_ENV" default:"development"`
	HTTPPort int      `envconfig:"HTTP_PORT" default:"8080"`
	LogLevel string   `envconfig:"LOG_LEVEL" default:"info"`
	Postgres Postgres `ignored:"true"`
	Redis    Redis    `ignored:"true"`
}

type Postgres struct {
	Host     string `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port     int    `envconfig:"POSTGRES_PORT" default:"5433"`
	User     string `envconfig:"POSTGRES_USER" default:"ecommerce"`
	Password string `envconfig:"POSTGRES_PASSWORD" default:"ecommerce"`
	DB       string `envconfig:"POSTGRES_DB" default:"ecommerce"`
	SSLMode  string `envconfig:"POSTGRES_SSLMODE" default:"disable"`
}

type Redis struct {
	Addr     string `envconfig:"REDIS_ADDR" default:"localhost:6380"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB" default:"0"`
}

func Load() (Config, error) {
	var cfg Config
	for _, target := range []any{&cfg, &cfg.Postgres, &cfg.Redis} {
		if err := envconfig.Process("", target); err != nil {
			return Config{}, fmt.Errorf("load config: %w", err)
		}
	}
	return cfg, nil
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func (p Postgres) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:     p.DB,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}
	return u.String()
}
