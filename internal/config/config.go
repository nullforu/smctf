package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
	Cache    CacheConfig
	Logging  LoggingConfig
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

type CacheConfig struct {
	TimelineTTL time.Duration
}

type LoggingConfig struct {
	Dir               string
	FilePrefix        string
	DiscordWebhookURL string
	SlackWebhookURL   string
	MaxBodyBytes      int
	WebhookQueueSize  int
	WebhookTimeout    time.Duration
	WebhookBatchSize  int
	WebhookBatchWait  time.Duration
	WebhookMaxChars   int
}

const (
	defaultJWTSecret  = "change-me"
	defaultFlagSecret = "change-me-too"
)

func Load() (Config, error) {
	var errs []error

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, fmt.Errorf("load .env: %w", err))
	}

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

	timelineCacheTTL, err := getDuration("TIMELINE_CACHE_TTL", 60*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	logDir := getEnv("LOG_DIR", "logs")
	logPrefix := getEnv("LOG_FILE_PREFIX", "app")
	logMaxBodyBytes, err := getEnvInt("LOG_MAX_BODY_BYTES", 1024*1024)
	if err != nil {
		errs = append(errs, err)
	}

	logWebhookQueueSize, err := getEnvInt("LOG_WEBHOOK_QUEUE_SIZE", 1000)
	if err != nil {
		errs = append(errs, err)
	}

	logWebhookTimeout, err := getDuration("LOG_WEBHOOK_TIMEOUT", 5*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	logWebhookBatchSize, err := getEnvInt("LOG_WEBHOOK_BATCH_SIZE", 20)
	if err != nil {
		errs = append(errs, err)
	}

	logWebhookBatchWait, err := getDuration("LOG_WEBHOOK_BATCH_WAIT", 2*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	logWebhookMaxChars, err := getEnvInt("LOG_WEBHOOK_MAX_CHARS", 1800)
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
		Cache: CacheConfig{
			TimelineTTL: timelineCacheTTL,
		},
		Logging: LoggingConfig{
			Dir:               logDir,
			FilePrefix:        logPrefix,
			DiscordWebhookURL: getEnv("LOG_DISCORD_WEBHOOK_URL", ""),
			SlackWebhookURL:   getEnv("LOG_SLACK_WEBHOOK_URL", ""),
			MaxBodyBytes:      logMaxBodyBytes,
			WebhookQueueSize:  logWebhookQueueSize,
			WebhookTimeout:    logWebhookTimeout,
			WebhookBatchSize:  logWebhookBatchSize,
			WebhookBatchWait:  logWebhookBatchWait,
			WebhookMaxChars:   logWebhookMaxChars,
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

	if cfg.Logging.Dir == "" {
		errs = append(errs, errors.New("LOG_DIR must not be empty"))
	}

	if cfg.Logging.FilePrefix == "" {
		errs = append(errs, errors.New("LOG_FILE_PREFIX must not be empty"))
	}

	if cfg.Logging.MaxBodyBytes <= 0 {
		errs = append(errs, errors.New("LOG_MAX_BODY_BYTES must be positive"))
	}

	if cfg.Logging.WebhookQueueSize <= 0 {
		errs = append(errs, errors.New("LOG_WEBHOOK_QUEUE_SIZE must be positive"))
	}

	if cfg.Logging.WebhookTimeout <= 0 {
		errs = append(errs, errors.New("LOG_WEBHOOK_TIMEOUT must be positive"))
	}

	if cfg.Logging.WebhookBatchSize <= 0 {
		errs = append(errs, errors.New("LOG_WEBHOOK_BATCH_SIZE must be positive"))
	}

	if cfg.Logging.WebhookBatchWait <= 0 {
		errs = append(errs, errors.New("LOG_WEBHOOK_BATCH_WAIT must be positive"))
	}

	if cfg.Logging.WebhookMaxChars <= 0 {
		errs = append(errs, errors.New("LOG_WEBHOOK_MAX_CHARS must be positive"))
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func Redact(cfg Config) Config {
	cfg.DB.Password = redact(cfg.DB.Password)
	cfg.Redis.Password = redact(cfg.Redis.Password)
	cfg.JWT.Secret = redact(cfg.JWT.Secret)
	cfg.Security.FlagHMACSecret = redact(cfg.Security.FlagHMACSecret)
	cfg.Logging.DiscordWebhookURL = redact(cfg.Logging.DiscordWebhookURL)
	cfg.Logging.SlackWebhookURL = redact(cfg.Logging.SlackWebhookURL)
	return cfg
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	const (
		visiblePrefix = 2
		visibleSuffix = 2
	)
	if len(value) <= visiblePrefix+visibleSuffix {
		return "***"
	}
	return value[:visiblePrefix] + "***" + value[len(value)-visibleSuffix:]
}

func FormatForLog(cfg Config) string {
	cfg = Redact(cfg)
	var b strings.Builder
	fmt.Fprintf(&b, "AppEnv=%s\n", cfg.AppEnv)
	fmt.Fprintf(&b, "HTTPAddr=%s\n", cfg.HTTPAddr)
	fmt.Fprintf(&b, "ShutdownTimeout=%s\n", cfg.ShutdownTimeout)
	fmt.Fprintf(&b, "AutoMigrate=%t\n", cfg.AutoMigrate)
	fmt.Fprintf(&b, "PasswordBcryptCost=%d\n", cfg.PasswordBcryptCost)
	fmt.Fprintln(&b, "DB:")
	fmt.Fprintf(&b, "  Host=%s\n", cfg.DB.Host)
	fmt.Fprintf(&b, "  Port=%d\n", cfg.DB.Port)
	fmt.Fprintf(&b, "  User=%s\n", cfg.DB.User)
	fmt.Fprintf(&b, "  Password=%s\n", cfg.DB.Password)
	fmt.Fprintf(&b, "  Name=%s\n", cfg.DB.Name)
	fmt.Fprintf(&b, "  SSLMode=%s\n", cfg.DB.SSLMode)
	fmt.Fprintf(&b, "  MaxOpenConns=%d\n", cfg.DB.MaxOpenConns)
	fmt.Fprintf(&b, "  MaxIdleConns=%d\n", cfg.DB.MaxIdleConns)
	fmt.Fprintf(&b, "  ConnMaxLifetime=%s\n", cfg.DB.ConnMaxLifetime)
	fmt.Fprintln(&b, "Redis:")
	fmt.Fprintf(&b, "  Addr=%s\n", cfg.Redis.Addr)
	fmt.Fprintf(&b, "  Password=%s\n", cfg.Redis.Password)
	fmt.Fprintf(&b, "  DB=%d\n", cfg.Redis.DB)
	fmt.Fprintf(&b, "  PoolSize=%d\n", cfg.Redis.PoolSize)
	fmt.Fprintln(&b, "JWT:")
	fmt.Fprintf(&b, "  Secret=%s\n", cfg.JWT.Secret)
	fmt.Fprintf(&b, "  Issuer=%s\n", cfg.JWT.Issuer)
	fmt.Fprintf(&b, "  AccessTTL=%s\n", cfg.JWT.AccessTTL)
	fmt.Fprintf(&b, "  RefreshTTL=%s\n", cfg.JWT.RefreshTTL)
	fmt.Fprintln(&b, "Security:")
	fmt.Fprintf(&b, "  FlagHMACSecret=%s\n", cfg.Security.FlagHMACSecret)
	fmt.Fprintf(&b, "  SubmissionWindow=%s\n", cfg.Security.SubmissionWindow)
	fmt.Fprintf(&b, "  SubmissionMax=%d\n", cfg.Security.SubmissionMax)
	fmt.Fprintln(&b, "Cache:")
	fmt.Fprintf(&b, "  TimelineTTL=%s\n", cfg.Cache.TimelineTTL)
	fmt.Fprintln(&b, "Logging:")
	fmt.Fprintf(&b, "  Dir=%s\n", cfg.Logging.Dir)
	fmt.Fprintf(&b, "  FilePrefix=%s\n", cfg.Logging.FilePrefix)
	fmt.Fprintf(&b, "  DiscordWebhookURL=%s\n", cfg.Logging.DiscordWebhookURL)
	fmt.Fprintf(&b, "  SlackWebhookURL=%s\n", cfg.Logging.SlackWebhookURL)
	fmt.Fprintf(&b, "  MaxBodyBytes=%d\n", cfg.Logging.MaxBodyBytes)
	fmt.Fprintf(&b, "  WebhookQueueSize=%d\n", cfg.Logging.WebhookQueueSize)
	fmt.Fprintf(&b, "  WebhookTimeout=%s\n", cfg.Logging.WebhookTimeout)
	fmt.Fprintf(&b, "  WebhookBatchSize=%d\n", cfg.Logging.WebhookBatchSize)
	fmt.Fprintf(&b, "  WebhookBatchWait=%s\n", cfg.Logging.WebhookBatchWait)
	fmt.Fprintf(&b, "  WebhookMaxChars=%d\n", cfg.Logging.WebhookMaxChars)
	return b.String()
}
