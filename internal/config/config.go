package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
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

const (
	defaultJWTSecret  = "dev-secret-change-me"
	defaultFlagSecret = "dev-flag-secret-change-me"
)

func Load() (Config, error) {
	var errs []error

	appEnv := getEnv("APP_ENV", "local")
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	shutdownTimeout, err := getDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	autoMigrate, err := getEnvBool("AUTO_MIGRATE", true)
	if err != nil {
		errs = append(errs, err)
	}

	bcryptCost, err := getEnvInt("BCRYPT_COST", 12)
	if err != nil {
		errs = append(errs, err)
	}

	dbPort, err := getEnvInt("DB_PORT", 5432)
	if err != nil {
		errs = append(errs, err)
	}

	dbMaxOpen, err := getEnvInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		errs = append(errs, err)
	}

	dbMaxIdle, err := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		errs = append(errs, err)
	}

	dbConnMaxLifetime, err := getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		errs = append(errs, err)
	}

	redisPoolSize, err := getEnvInt("REDIS_POOL_SIZE", 20)
	if err != nil {
		errs = append(errs, err)
	}

	jwtAccessTTL, err := getDuration("JWT_ACCESS_TTL", 24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}

	jwtRefreshTTL, err := getDuration("JWT_REFRESH_TTL", 7*24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}

	submitWindow, err := getDuration("SUBMIT_WINDOW", 1*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	submitMax, err := getEnvInt("SUBMIT_MAX", 10)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := Config{
		AppEnv:             appEnv,
		HTTPAddr:           httpAddr,
		ShutdownTimeout:    shutdownTimeout,
		AutoMigrate:        autoMigrate,
		PasswordBcryptCost: bcryptCost,
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            dbPort,
			User:            getEnv("DB_USER", "app_user"),
			Password:        getEnv("DB_PASSWORD", "app_password"),
			Name:            getEnv("DB_NAME", "app_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    dbMaxOpen,
			MaxIdleConns:    dbMaxIdle,
			ConnMaxLifetime: dbConnMaxLifetime,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
			PoolSize: redisPoolSize,
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", defaultJWTSecret),
			Issuer:     getEnv("JWT_ISSUER", "smctf"),
			AccessTTL:  jwtAccessTTL,
			RefreshTTL: jwtRefreshTTL,
		},
		Security: SecurityConfig{
			FlagHMACSecret:   getEnv("FLAG_HMAC_SECRET", defaultFlagSecret),
			SubmissionWindow: submitWindow,
			SubmissionMax:    submitMax,
		},
	}

	if err := validateConfig(cfg); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}

	return v
}

func getEnvInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("%s must be an integer", key)
	}

	return n, nil
}

func getEnvBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, fmt.Errorf("%s must be a boolean", key)
	}

	return b, nil
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return def, fmt.Errorf("%s must be a duration", key)
	}

	return d, nil
}

func validateConfig(cfg Config) error {
	var errs []error

	if cfg.HTTPAddr == "" {
		errs = append(errs, errors.New("HTTP_ADDR must not be empty"))
	}

	if cfg.PasswordBcryptCost < bcrypt.MinCost || cfg.PasswordBcryptCost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf("BCRYPT_COST must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost))
	}

	// Database validation
	if cfg.DB.Host == "" || cfg.DB.Name == "" || cfg.DB.User == "" {
		errs = append(errs, errors.New("DB_HOST, DB_NAME, and DB_USER must be set"))
	}
	if cfg.DB.Port <= 0 {
		errs = append(errs, errors.New("DB_PORT must be a positive integer"))
	}
	if cfg.DB.MaxOpenConns <= 0 || cfg.DB.MaxIdleConns <= 0 {
		errs = append(errs, errors.New("DB_MAX_OPEN_CONNS and DB_MAX_IDLE_CONNS must be positive"))
	}
	if cfg.DB.ConnMaxLifetime <= 0 {
		errs = append(errs, errors.New("DB_CONN_MAX_LIFETIME must be positive"))
	}

	// Redis validation
	if cfg.Redis.Addr == "" {
		errs = append(errs, errors.New("REDIS_ADDR must not be empty"))
	}
	if cfg.Redis.PoolSize <= 0 {
		errs = append(errs, errors.New("REDIS_POOL_SIZE must be positive"))
	}

	// JWT validation
	if cfg.JWT.Secret == "" {
		errs = append(errs, errors.New("JWT_SECRET must not be empty"))
	}
	if cfg.JWT.Issuer == "" {
		errs = append(errs, errors.New("JWT_ISSUER must not be empty"))
	}
	if cfg.JWT.AccessTTL <= 0 || cfg.JWT.RefreshTTL <= 0 {
		errs = append(errs, errors.New("JWT_ACCESS_TTL and JWT_REFRESH_TTL must be positive"))
	}

	// Security validation
	if cfg.Security.FlagHMACSecret == "" {
		errs = append(errs, errors.New("FLAG_HMAC_SECRET must not be empty"))
	}
	if cfg.Security.SubmissionWindow <= 0 || cfg.Security.SubmissionMax <= 0 {
		errs = append(errs, errors.New("SUBMIT_WINDOW and SUBMIT_MAX must be positive"))
	}

	// Production-specific validation
	if cfg.AppEnv == "production" {
		if cfg.JWT.Secret == defaultJWTSecret {
			errs = append(errs, errors.New("JWT_SECRET must be set in production"))
		}
		if cfg.Security.FlagHMACSecret == defaultFlagSecret {
			errs = append(errs, errors.New("FLAG_HMAC_SECRET must be set in production"))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
