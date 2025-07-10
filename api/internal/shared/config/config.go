package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	NATS       NATSConfig       `mapstructure:"nats"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Stripe     StripeConfig     `mapstructure:"stripe"`
	K8s        K8sConfig        `mapstructure:"k8s"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	AIOps      AIOpsConfig      `mapstructure:"aiops"`
	ClickHouse ClickHouseConfig `mapstructure:"clickhouse"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL       string `mapstructure:"url"`
	ClusterID string `mapstructure:"cluster_id"`
	ClientID  string `mapstructure:"client_id"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret         string                   `mapstructure:"jwt_secret"`
	JWTExpiration     int                      `mapstructure:"jwt_expiration"`
	ExternalProviders map[string]OAuthProvider `mapstructure:"external_providers"`
}

// OAuthProvider holds OAuth provider configuration
type OAuthProvider struct {
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
	AuthURL      string   `mapstructure:"auth_url"`
	TokenURL     string   `mapstructure:"token_url"`
	UserInfoURL  string   `mapstructure:"userinfo_url"`
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	APIKey            string `mapstructure:"api_key"`
	SecretKey         string `mapstructure:"secret_key"`
	PublishableKey    string `mapstructure:"publishable_key"`
	WebhookSecret     string `mapstructure:"webhook_secret"`
	PriceIDBasic      string `mapstructure:"price_id_basic"`
	PriceIDPro        string `mapstructure:"price_id_pro"`
	PriceIDEnterprise string `mapstructure:"price_id_enterprise"`
}

// K8sConfig holds Kubernetes configuration
type K8sConfig struct {
	ConfigPath        string `mapstructure:"config_path"`
	InCluster         bool   `mapstructure:"in_cluster"`
	VClusterNamespace string `mapstructure:"vcluster_namespace"`
}

// MonitoringConfig holds monitoring and metrics configuration
type MonitoringConfig struct {
	PrometheusURL     string   `mapstructure:"prometheus_url"`
	MetricsPort       string   `mapstructure:"metrics_port"`
	MetricsPath       string   `mapstructure:"metrics_path"`
	ScrapeInterval    string   `mapstructure:"scrape_interval"`
	RetentionPeriod   string   `mapstructure:"retention_period"`
	EnableMetrics     bool     `mapstructure:"enable_metrics"`
	EnableAlerts      bool     `mapstructure:"enable_alerts"`
	AlertmanagerURL   string   `mapstructure:"alertmanager_url"`
	DefaultAlertRules []string `mapstructure:"default_alert_rules"`
}

// AIOpsConfig holds configuration for the AIOps service
type AIOpsConfig struct {
	URL string `mapstructure:"url"`
}

// ClickHouseConfig holds configuration for the ClickHouse database.
type ClickHouseConfig struct {
	Address  string `mapstructure:"address"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/hexabase")
	}

	// Set defaults
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 120)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.dbname", "hexabase")
	viper.SetDefault("database.sslmode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "hexabase-cluster")
	viper.SetDefault("nats.client_id", "hexabase-api")

	// Auth defaults
	viper.SetDefault("auth.jwt_expiration", 3600) // 1 hour

	// AIOps defaults
	viper.SetDefault("aiops.url", "http://ai-ops-service.ai-ops.svc.cluster.local:8000")

	// ClickHouse defaults
	viper.SetDefault("clickhouse.address", "llmops-clickhouse.llm-ops.svc.cluster.local:9000")
	viper.SetDefault("clickhouse.user", "default")
	viper.SetDefault("clickhouse.password", "")
	viper.SetDefault("clickhouse.database", "default")

	// K8s defaults
	viper.SetDefault("k8s.in_cluster", false)
	viper.SetDefault("k8s.vcluster_namespace", "vcluster")

	// Monitoring defaults
	viper.SetDefault("monitoring.prometheus_url", "http://localhost:9090")
	viper.SetDefault("monitoring.metrics_port", "2112")
	viper.SetDefault("monitoring.metrics_path", "/metrics")
	viper.SetDefault("monitoring.scrape_interval", "15s")
	viper.SetDefault("monitoring.retention_period", "15d")
	viper.SetDefault("monitoring.enable_metrics", true)
	viper.SetDefault("monitoring.enable_alerts", true)
	viper.SetDefault("monitoring.alertmanager_url", "http://localhost:9093")
	viper.SetDefault("monitoring.default_alert_rules", []string{"high_cpu", "high_memory", "pod_restarts"})
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if c.Stripe.SecretKey == "" {
		return fmt.Errorf("Stripe secret key is required")
	}

	return nil
}
