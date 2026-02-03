package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Database    DatabaseConfig    `mapstructure:"database"`
	AI          AIConfig          `mapstructure:"ai"`
	Application ApplicationConfig `mapstructure:"application"`
	Ldap        LdapConfig        `mapstructure:"ldap"`
}

type ApplicationConfig struct {
	Name               string        `mapstructure:"name"`
	Version            string        `mapstructure:"version"`
	LastBuild          string        `mapstructure:"last_build"`
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	Author             string        `mapstructure:"author"`
	Copyright          string        `mapstructure:"copyright"`
	Engine             string        `mapstructure:"engine"`
	ReferenceDocPath   string        `mapstructure:"reference_doc_path"`
	ParserRulesDocPath string        `mapstructure:"parser_rules_doc_path"`
	Storage            StorageConfig `mapstructure:"storage"`
	Authentication     bool          `mapstructure:"authentication"`
	AuthType           string        `mapstructure:"auth_type"`
}

type StorageConfig struct {
	// Original deprecated, use Stage
	Original   string `mapstructure:"original"`
	Thumbnails string `mapstructure:"thumbnails"`
	Stage      string `mapstructure:"stage"`
	Template   string `mapstructure:"template"`
}

type AIConfig struct {
	ActiveProvider string                      `mapstructure:"active_provider"`
	Providers      map[string]ProviderSettings `mapstructure:"providers"`
}

type ProviderSettings struct {
	Driver      string  `mapstructure:"driver"` // gemini, openai, anthropic
	Key         string  `mapstructure:"key"`
	Endpoint    string  `mapstructure:"endpoint"`
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

type LdapConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	BindDN       string `mapstructure:"bind_dn"`
	Password     string `mapstructure:"password"`
	BaseDN       string `mapstructure:"base_dn"`
	Prefix       string `mapstructure:"prefix"`
	Postfix      string `mapstructure:"postfix"`
	SearchFilter string `mapstructure:"search_filter"`
}

func (c *LdapConfig) IsAuthEnabled() bool {
	return c.Enabled
}

type DatabaseConfig struct {
	URL      string `mapstructure:"url"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
	Options  string `mapstructure:"options"`
}

func (c *DatabaseConfig) GetConnectStr() string {
	if c.URL != "" {
		return c.URL
	}
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, sslmode)

	if c.Options != "" {
		// Basic URL encoding for the options value: space -> %20
		encodedOptions := strings.ReplaceAll(c.Options, " ", "%20")
		connStr += fmt.Sprintf("&options=%s", encodedOptions)
	}

	return connStr
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Note: .env file not found, using system environment variables")
	}

	viper.SetConfigFile("config.yaml") // Support optional config.yaml
	viper.AutomaticEnv()

	// Environment variable mappings
	mappings := []struct {
		key, env string
	}{
		{"database.url", "DB_URL"},
		{"database.host", "PG_HOST"},
		{"database.port", "PG_PORT"},
		{"database.user", "PG_USER"},
		{"database.password", "PG_PASSWORD"},
		{"database.dbname", "PG_DB"},
		{"database.sslmode", "PG_SSLMODE"},
		{"database.options", "PG_OPTIONS"},
		{"application.port", "PORT"},
		{"application.authentication", "AUTH_ENABLED"},
		{"application.auth_type", "AUTH_TYPE"},
		{"ai.active_provider", "AI_PROVIDER"},

		// Storage
		{"application.storage.stage", "STORAGE_STAGE"},
		{"application.storage.template", "STORAGE_TEMPLATE"},
		{"application.storage.thumbnails", "STORAGE_THUMBNAILS"},

		// AI Providers
		{"ai.providers.gemini.key", "GEMINI_KEY"},
		{"ai.providers.gemini.model", "GEMINI_MODEL"},
		{"ai.providers.openai.key", "OPENAI_API_KEY"},
		{"ai.providers.openai.model", "OPENAI_MODEL"},
		{"ai.providers.claude.key", "ANTHROPIC_API_KEY"},
		{"ai.providers.claude.model", "CLAUDE_MODEL"},

		// LDAP/AD
		{"ldap.enabled", "LDAP_ENABLED"},
		{"ldap.host", "LDAP_HOST"},
		{"ldap.port", "LDAP_PORT"},
		{"ldap.bind_dn", "LDAP_BIND_DN"},
		{"ldap.password", "LDAP_PASSWORD"},
		{"ldap.base_dn", "LDAP_BASE_DN"},
		{"ldap.prefix", "LDAP_PREFIX"},
		{"ldap.postfix", "LDAP_POSTFIX"},
		{"ldap.search_filter", "LDAP_SEARCH_FILTER"},
	}

	for _, m := range mappings {
		viper.BindEnv(m.key, m.env)
	}
	viper.BindEnv("application.authentication", "AUTHENTICATION")

	// Defaults
	viper.SetDefault("application.authentication", true)
	viper.SetDefault("ldap.enabled", false)
	viper.SetDefault("ldap.host", "ldap.alig.hu")
	viper.SetDefault("ldap.port", 389)
	viper.SetDefault("ldap.search_filter", "(&(objectCategory=person)(objectClass=user))")
	viper.SetDefault("ldap.postfix", "@alig.hu")

	if err := viper.ReadInConfig(); err != nil {
		// Ignore if config.yaml is missing
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.AI.ActiveProvider == "" {
		cfg.AI.ActiveProvider = "gemini"
	}

	return &cfg, nil
}
