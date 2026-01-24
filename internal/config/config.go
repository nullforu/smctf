package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	ShutdownTimeout    time.Duration
	AutoMigrate        bool
	PasswordBcryptCost int

	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Security SecurityConfig
}

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type JWTConfig struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type SecurityConfig struct {
	FlagHMACSecret   string
	SubmissionWindow time.Duration
	SubmissionMax    int
}

func Load() Config {
	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		ShutdownTimeout:    getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		AutoMigrate:        getEnvBool("AUTO_MIGRATE", true),
		PasswordBcryptCost: getEnvInt("BCRYPT_COST", 12),
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "app_user"),
			Password:        getEnv("DB_PASSWORD", "app_password"),
			Name:            getEnv("DB_NAME", "app_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 20),
		},
		JWT: JWTConfig{
			Secret:     mustEnv("JWT_SECRET", "dev-secret-change-me"),
			Issuer:     getEnv("JWT_ISSUER", "smctf"),
			AccessTTL:  getDuration("JWT_ACCESS_TTL", 24*time.Hour),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Security: SecurityConfig{
			FlagHMACSecret:   mustEnv("FLAG_HMAC_SECRET", "dev-flag-secret-change-me"),
			SubmissionWindow: getDuration("SUBMIT_WINDOW", 1*time.Minute),
			SubmissionMax:    getEnvInt("SUBMIT_MAX", 10),
		},
	}
	return cfg
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func mustEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
