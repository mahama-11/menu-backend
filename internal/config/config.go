package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Host       string           `mapstructure:"host"`
	Port       int              `mapstructure:"port"`
	GinMode    string           `mapstructure:"gin_mode"`
	LogLevel   string           `mapstructure:"log_level"`
	UseMock    bool             `mapstructure:"use_mock"`
	App        AppConfig        `mapstructure:"app"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Security   SecurityConfig   `mapstructure:"security"`
	Platform   PlatformConfig   `mapstructure:"platform"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

type AppConfig struct {
	FrontendBaseURL    string `mapstructure:"frontend_base_url"`
	SignupBonusCredits int64  `mapstructure:"signup_bonus_credits"`
	CreditsAssetCode   string `mapstructure:"credits_asset_code"`
}

type DatabaseConfig struct {
	Driver             string `mapstructure:"driver"`
	Host               string `mapstructure:"host"`
	Port               int    `mapstructure:"port"`
	User               string `mapstructure:"user"`
	Password           string `mapstructure:"password"`
	DBName             string `mapstructure:"dbname"`
	SSLMode            string `mapstructure:"sslmode"`
	MaxOpenConns       int    `mapstructure:"max_open_conns"`
	MaxIdleConns       int    `mapstructure:"max_idle_conns"`
	SQLitePath         string `mapstructure:"sqlite_path"`
	TablePrefix        string `mapstructure:"table_prefix"`
	AutoMigrateEnabled bool   `mapstructure:"auto_migrate_enabled"`
}

type RedisConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	MaxRetries   int           `mapstructure:"max_retries"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type SecurityConfig struct {
	JWTSecret        string `mapstructure:"jwt_secret"`
	EncryptionKey    string `mapstructure:"encryption_key"`
	ServiceSecretKey string `mapstructure:"service_secret_key"`
}

type PlatformConfig struct {
	BaseURL               string        `mapstructure:"base_url"`
	Timeout               time.Duration `mapstructure:"timeout"`
	ServiceName           string        `mapstructure:"service_name"`
	InternalServiceSecret string        `mapstructure:"internal_service_secret"`
	JWTSecret             string        `mapstructure:"jwt_secret"`
}

type MonitoringConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
}

type MetricsConfig struct {
	Enabled          bool      `mapstructure:"enabled"`
	Port             int       `mapstructure:"port"`
	Path             string    `mapstructure:"path"`
	Namespace        string    `mapstructure:"namespace"`
	Subsystem        string    `mapstructure:"subsystem"`
	PushInterval     string    `mapstructure:"push_interval"`
	HistogramBuckets []float64 `mapstructure:"histogram_buckets"`
}

type TracingConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
	Environment    string  `mapstructure:"environment"`
	JaegerEndpoint string  `mapstructure:"jaeger_endpoint"`
	SampleRate     float64 `mapstructure:"sample_rate"`
	LogSpans       bool    `mapstructure:"log_spans"`
}

func Load(configFile string) (*Config, error) {
	if configFile == "" {
		configFile = "config.local"
	}
	v := viper.New()
	v.SetConfigName(strings.TrimSuffix(configFile, ".yaml"))
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/menu-service/")
	v.SetEnvPrefix("MENU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("host", "0.0.0.0")
	v.SetDefault("port", 8096)
	v.SetDefault("gin_mode", "debug")
	v.SetDefault("log_level", "info")
	v.SetDefault("use_mock", false)
	v.SetDefault("app.frontend_base_url", "http://localhost:5173")
	v.SetDefault("app.signup_bonus_credits", 20)
	v.SetDefault("app.credits_asset_code", "MENU_CREDIT")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.host", "database")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "menu")
	v.SetDefault("database.password", "menupassword")
	v.SetDefault("database.dbname", "menu")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.sqlite_path", "data/menu.db")
	v.SetDefault("database.table_prefix", "menu_")
	v.SetDefault("database.auto_migrate_enabled", true)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.host", "redis")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 1)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("security.jwt_secret", "menu-dev-secret")
	v.SetDefault("security.encryption_key", "menu-encryption-key-change-me")
	v.SetDefault("security.service_secret_key", "menu-service-secret")
	v.SetDefault("platform.base_url", "http://v-platform-backend:8095")
	v.SetDefault("platform.timeout", "5s")
	v.SetDefault("platform.service_name", "v-menu-backend")
	v.SetDefault("platform.internal_service_secret", "platform-internal-secret")
	v.SetDefault("platform.jwt_secret", "platform-dev-secret")
	v.SetDefault("monitoring.metrics.enabled", true)
	v.SetDefault("monitoring.metrics.port", 9092)
	v.SetDefault("monitoring.metrics.path", "/metrics")
	v.SetDefault("monitoring.metrics.namespace", "menu")
	v.SetDefault("monitoring.metrics.subsystem", "service")
	v.SetDefault("monitoring.metrics.push_interval", "30s")
	v.SetDefault("monitoring.metrics.histogram_buckets", []float64{0.1, 0.5, 1, 2, 5, 10})
	v.SetDefault("monitoring.tracing.enabled", false)
	v.SetDefault("monitoring.tracing.service_name", "menu-service")
	v.SetDefault("monitoring.tracing.service_version", "1.0.0")
	v.SetDefault("monitoring.tracing.environment", "development")
	v.SetDefault("monitoring.tracing.jaeger_endpoint", "http://localhost:14268/api/traces")
	v.SetDefault("monitoring.tracing.sample_rate", 1.0)
	v.SetDefault("monitoring.tracing.log_spans", false)
}
