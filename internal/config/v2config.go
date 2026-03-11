package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Config represents the full V2 configuration
type Config struct {
	Version         string `json:"version"`
	SetupComplete   bool   `json:"setup_complete"`
	DataDir         string `json:"data_dir"`
	LogLevel        string `json:"log_level"`
	AutoOpenBrowser bool   `json:"auto_open_browser"`
	CheckForUpdates bool   `json:"check_for_updates"`
	CesiumToken     string `json:"cesium_token"`

	Server          ServerConfig          `json:"server"`
	Telegram        TelegramConfig        `json:"telegram"`
	Slack           SlackConfig           `json:"slack"`
	Discord         DiscordConfig         `json:"discord"`
	Ntfy            NtfyConfig            `json:"ntfy"`
	Pushover        PushoverConfig        `json:"pushover"`
	Email           EmailConfig           `json:"email"`
	Keys            KeysConfig            `json:"keys"`
	Providers       ProvidersConfig       `json:"providers"`
	Notifications   NotificationsConfig   `json:"notifications"`
	MorningBriefing MorningBriefingConfig `json:"morning_briefing"`
	WeeklyDigest    WeeklyDigestConfig    `json:"weekly_digest"`
	UI              UIConfig              `json:"ui"`
	Location        LocationConfig        `json:"location"`

	// V3 additions
	SignalBoard    SignalBoardConfig    `json:"signal_board,omitempty"`
	EntityTracking EntityTrackingConfig `json:"entity_tracking,omitempty"`
}

type ServerConfig struct {
	Port             int    `json:"port"`
	Host             string `json:"host"`
	TLSEnabled       bool   `json:"tls_enabled"`
	TLSCert          string `json:"tls_cert"`
	TLSKey           string `json:"tls_key"`
	AuthEnabled      bool   `json:"auth_enabled"`
	AuthToken        string `json:"auth_token"`
	DashboardPassword string `json:"dashboard_password"`
}

type TelegramConfig struct {
	Enabled              bool   `json:"enabled"`
	BotToken             string `json:"bot_token"`
	ChatID               string `json:"chat_id"`
	MinSeverity          string `json:"min_severity"`
	DigestMode           bool   `json:"digest_mode"`
	DigestIntervalMinutes int    `json:"digest_interval_minutes"`
}

type SlackConfig struct {
	Enabled     bool   `json:"enabled"`
	WebhookURL  string `json:"webhook_url"`
	Channel     string `json:"channel"`
	MinSeverity string `json:"min_severity"`
}

type DiscordConfig struct {
	Enabled     bool   `json:"enabled"`
	WebhookURL  string `json:"webhook_url"`
	MinSeverity string `json:"min_severity"`
}

type NtfyConfig struct {
	Enabled     bool   `json:"enabled"`
	Server      string `json:"server"`
	Topic       string `json:"topic"`
	MinSeverity string `json:"min_severity"`
}

type PushoverConfig struct {
	Enabled  bool   `json:"enabled"`
	AppToken string `json:"app_token"`
	UserKey  string `json:"user_key"`
}

type EmailConfig struct {
	Enabled               bool     `json:"enabled"`
	Method                string   `json:"method"`
	SMTPHost              string   `json:"smtp_host"`
	SMTPPort              int      `json:"smtp_port"`
	SMTPTLS               string   `json:"smtp_tls"`
	Username              string   `json:"username"`
	PasswordEncrypted     string   `json:"password_encrypted"`
	FromAddress           string   `json:"from_address"`
	ToAddresses           []string `json:"to_addresses"`
	GmailClientID         string   `json:"gmail_client_id"`
	GmailClientSecret     string   `json:"gmail_client_secret"`
	GmailRefreshToken     string   `json:"gmail_refresh_token"`
	SendgridKeyEncrypted  string   `json:"sendgrid_key_encrypted"`
	MailgunKeyEncrypted   string   `json:"mailgun_key_encrypted"`
	MailgunDomain         string   `json:"mailgun_domain"`
	MinSeverity           string   `json:"min_severity"`
}

type KeysConfig struct {
	Adsbexchange   string `json:"adsbexchange"`
	Aisstream      string `json:"aisstream"`
	Acled          string `json:"acled"`
	Openweather    string `json:"openweather"`
	Nasa           string `json:"nasa"`
	Spacetrack     string `json:"spacetrack"`
	Marinetraffic  string `json:"marinetraffic"`
	Vesselfinder   string `json:"vesselfinder"`
	N2yo           string `json:"n2yo"`
	Shodan         string `json:"shodan"`
	Cloudflare     string `json:"cloudflare"`
	Ukrainealerts  string `json:"ukrainealerts"`
	AlphaVantage   string `json:"alpha_vantage"`
	Finnhub        string `json:"finnhub"`
	Fred           string `json:"fred"`
	Polygon        string `json:"polygon"`
}

type ProviderConfig struct {
	Enabled         bool              `json:"enabled"`
	IntervalSeconds int               `json:"interval_seconds"`
	Options         map[string]string `json:"options,omitempty"`
}

type ProvidersConfig struct {
	USGS           ProviderConfig `json:"usgs"`
	GDACS          ProviderConfig `json:"gdacs"`
	OpenSky        ProviderConfig `json:"opensky"`
	NOAACAP        ProviderConfig `json:"noaa_cap"`
	OpenMeteo      ProviderConfig `json:"openmeteo"`
	GDELT          ProviderConfig `json:"gdelt"`
	Celestrak      ProviderConfig `json:"celestrak"`
	SWPC           ProviderConfig `json:"swpc"`
	WHO            ProviderConfig `json:"who"`
	ProMED         ProviderConfig `json:"promed"`
	AirplanesLive  ProviderConfig `json:"airplanes_live"`
	NASAFIRMS      ProviderConfig `json:"nasa_firms"`
	PiracyIMB      ProviderConfig `json:"piracy_imb"`
	IsraelAlerts   ProviderConfig `json:"israel_alerts"`
	ReliefWeb      ProviderConfig `json:"reliefweb"`
	VIX            ProviderConfig `json:"vix"`
	OilPrice       ProviderConfig `json:"oil_price"`
	Crypto         ProviderConfig `json:"crypto"`
	SECEdgar       ProviderConfig `json:"sec_edgar"`
	OFACSDN        ProviderConfig `json:"ofac_sdn"`
	TreasuryYields ProviderConfig `json:"treasury_yields"`
	NewsRSS        ProviderConfig `json:"news_rss"`
	IranConflict   ProviderConfig `json:"iran_conflict"`
	ISW            ProviderConfig `json:"isw"`
}

type NotificationRule struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Enabled   bool                   `json:"enabled"`
	Condition map[string]interface{} `json:"condition"`
	Actions   []string               `json:"actions"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type Geofence struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Type      string    `json:"type"` // "circle" or "polygon"
	CenterLat float64   `json:"center_lat,omitempty"`
	CenterLon float64   `json:"center_lon,omitempty"`
	RadiusKm  float64   `json:"radius_km,omitempty"`
	Coordinates [][]float64 `json:"coordinates,omitempty"` // for polygon
	CreatedAt time.Time `json:"created_at"`
}

type NotificationsConfig struct {
	Rules     []NotificationRule `json:"rules"`
	Geofences []Geofence         `json:"geofences"`
}

type MorningBriefingConfig struct {
	Enabled                bool     `json:"enabled"`
	TimeUTC                string   `json:"time_utc"`
	Delivery               []string `json:"delivery"`
	IncludeEvents          bool     `json:"include_events"`
	IncludeConflicts       bool     `json:"include_conflicts"`
	IncludeSpaceWeather    bool     `json:"include_space_weather"`
	IncludeFinancial       bool     `json:"include_financial"`
	IncludeISSPasses       bool     `json:"include_iss_passes"`
	IncludeNews            bool     `json:"include_news"`
}

type WeeklyDigestConfig struct {
	Enabled   bool     `json:"enabled"`
	Day       string   `json:"day"`
	TimeUTC   string   `json:"time_utc"`
	Delivery  []string `json:"delivery"`
}

type UIConfig struct {
	DefaultView          string `json:"default_view"`
	DefaultPreset        string `json:"default_preset"`
	DataRetentionDays    int    `json:"data_retention_days"`
	SoundEnabled         bool   `json:"sound_enabled"`
	SoundVolume          int    `json:"sound_volume"`
	TickerEnabled        bool   `json:"ticker_enabled"`
	TickerSpeed          string `json:"ticker_speed"`
	TickerMinSeverity    string `json:"ticker_min_severity"`
}

type LocationConfig struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	RadiusKm float64 `json:"radius_km,omitempty"` // proximity alert radius
	Timezone string  `json:"timezone"`
	Set      bool    `json:"set"`
}

// SignalBoardConfig controls the domain threat-level dashboard
type SignalBoardConfig struct {
	Enabled bool `json:"enabled"`
}

// EntityTrackingConfig controls aircraft/vessel dead-reckoning
type EntityTrackingConfig struct {
	Enabled           bool `json:"enabled"`
	DeadReckoningMins int  `json:"dead_reckoning_mins,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Version:         "2.0.0",
		SetupComplete:   false,
		DataDir:         "",
		LogLevel:        "info",
		AutoOpenBrowser: true,
		CheckForUpdates: true,
		CesiumToken:     "",
		
		Server: ServerConfig{
			Port:             8080,
			Host:             "0.0.0.0",
			TLSEnabled:       false,
			TLSCert:          "",
			TLSKey:           "",
			AuthEnabled:      false,
			AuthToken:        "",
			DashboardPassword: "",
		},
		
		Telegram: TelegramConfig{
			Enabled:              false,
			BotToken:             "",
			ChatID:               "",
			MinSeverity:          "warning",
			DigestMode:           false,
			DigestIntervalMinutes: 60,
		},
		
		Slack: SlackConfig{
			Enabled:     false,
			WebhookURL:  "",
			Channel:     "#sentinel-alerts",
			MinSeverity: "warning",
		},
		
		Discord: DiscordConfig{
			Enabled:     false,
			WebhookURL:  "",
			MinSeverity: "warning",
		},
		
		Ntfy: NtfyConfig{
			Enabled:     false,
			Server:      "https://ntfy.sh",
			Topic:       "",
			MinSeverity: "warning",
		},
		
		Pushover: PushoverConfig{
			Enabled:  false,
			AppToken: "",
			UserKey:  "",
		},
		
		Email: EmailConfig{
			Enabled:               false,
			Method:                "",
			SMTPHost:              "",
			SMTPPort:              587,
			SMTPTLS:               "starttls",
			Username:              "",
			PasswordEncrypted:     "",
			FromAddress:           "",
			ToAddresses:           []string{},
			GmailClientID:         "",
			GmailClientSecret:     "",
			GmailRefreshToken:     "",
			SendgridKeyEncrypted:  "",
			MailgunKeyEncrypted:   "",
			MailgunDomain:         "",
			MinSeverity:           "alert",
		},
		
		Keys: KeysConfig{
			Adsbexchange:   "",
			Aisstream:      "",
			Acled:          "",
			Openweather:    "",
			Nasa:           "",
			Spacetrack:     "",
			Marinetraffic:  "",
			Vesselfinder:   "",
			N2yo:           "",
			Shodan:         "",
			Cloudflare:     "",
			Ukrainealerts:  "",
			AlphaVantage:   "",
			Finnhub:        "",
			Fred:           "",
			Polygon:        "",
		},
		
		Providers: ProvidersConfig{
			USGS:           ProviderConfig{Enabled: true, IntervalSeconds: 60},
			GDACS:          ProviderConfig{Enabled: true, IntervalSeconds: 60},
			OpenSky:        ProviderConfig{Enabled: true, IntervalSeconds: 60},
			NOAACAP:        ProviderConfig{Enabled: true, IntervalSeconds: 300},
			OpenMeteo:      ProviderConfig{Enabled: true, IntervalSeconds: 600},
			GDELT:          ProviderConfig{Enabled: true, IntervalSeconds: 900},
			Celestrak:      ProviderConfig{Enabled: true, IntervalSeconds: 21600},
			SWPC:           ProviderConfig{Enabled: true, IntervalSeconds: 60},
			WHO:            ProviderConfig{Enabled: true, IntervalSeconds: 3600},
			ProMED:         ProviderConfig{Enabled: true, IntervalSeconds: 1800},
			AirplanesLive:  ProviderConfig{Enabled: true, IntervalSeconds: 30},
			NASAFIRMS:      ProviderConfig{Enabled: true, IntervalSeconds: 1800},
			PiracyIMB:      ProviderConfig{Enabled: true, IntervalSeconds: 3600},
			IsraelAlerts:   ProviderConfig{Enabled: true, IntervalSeconds: 5},
			ReliefWeb:      ProviderConfig{Enabled: true, IntervalSeconds: 600},
			VIX:            ProviderConfig{Enabled: true, IntervalSeconds: 60},
			OilPrice:       ProviderConfig{Enabled: true, IntervalSeconds: 300},
			Crypto:         ProviderConfig{Enabled: true, IntervalSeconds: 60},
			SECEdgar:       ProviderConfig{Enabled: true, IntervalSeconds: 900},
			OFACSDN:        ProviderConfig{Enabled: true, IntervalSeconds: 3600},
			TreasuryYields: ProviderConfig{Enabled: true, IntervalSeconds: 3600},
			NewsRSS:        ProviderConfig{Enabled: true, IntervalSeconds: 1800},
			IranConflict:   ProviderConfig{Enabled: true, IntervalSeconds: 900},  // 15 minutes
			ISW:            ProviderConfig{Enabled: true, IntervalSeconds: 1800}, // 30 minutes
		},
		
		Notifications: NotificationsConfig{
			Rules:     []NotificationRule{},
			Geofences: []Geofence{},
		},
		
		MorningBriefing: MorningBriefingConfig{
			Enabled:                false,
			TimeUTC:                "08:00",
			Delivery:               []string{"telegram"},
			IncludeEvents:          true,
			IncludeConflicts:       true,
			IncludeSpaceWeather:    true,
			IncludeFinancial:       true,
			IncludeISSPasses:       true,
			IncludeNews:            true,
		},
		
		WeeklyDigest: WeeklyDigestConfig{
			Enabled:   false,
			Day:       "sunday",
			TimeUTC:   "08:00",
			Delivery:  []string{"email"},
		},
		
		UI: UIConfig{
			DefaultView:          "globe",
			DefaultPreset:        "Global Watch",
			DataRetentionDays:    30,
			SoundEnabled:         true,
			SoundVolume:          75,
			TickerEnabled:        true,
			TickerSpeed:          "medium",
			TickerMinSeverity:    "warning",
		},
		
		Location: LocationConfig{
			Lat:      0.0,
			Lon:      0.0,
			Timezone: "UTC",
			Set:      false,
		},
	}
}

// GetDefaultDataDir returns the platform-specific default data directory
func GetDefaultDataDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "SENTINEL", "data")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "SENTINEL", "data")
	default: // linux and others
		return filepath.Join(os.Getenv("HOME"), ".local", "share", "sentinel")
	}
}

// GetDefaultConfigDir returns the platform-specific default config directory
func GetDefaultConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "SENTINEL")
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "SENTINEL")
	default: // linux and others
		return filepath.Join(os.Getenv("HOME"), ".config", "sentinel")
	}
}

// GetDefaultConfigPath returns the default config file path
func GetDefaultConfigPath() string {
	return filepath.Join(GetDefaultConfigDir(), "config.json")
}

// LoadConfig loads configuration from file or returns default
func LoadConfig(configPath string) (*Config, error) {
	var config *Config
	
	if configPath == "" {
		configPath = GetDefaultConfigPath()
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config
		config = DefaultConfig()
		config.DataDir = GetDefaultDataDir()
		return config, nil
	}
	
	// Load from file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	config = DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Ensure DataDir is set
	if config.DataDir == "" {
		config.DataDir = GetDefaultDataDir()
	}
	
	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	if configPath == "" {
		configPath = GetDefaultConfigPath()
	}
	
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Ensure data directory exists
	if config.DataDir != "" {
		if err := os.MkdirAll(config.DataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}
	
	// Marshal with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetDBPath returns the database path
func (c *Config) GetDBPath() string {
	if c.DataDir == "" {
		return "sentinel.db" // fallback
	}
	return filepath.Join(c.DataDir, "sentinel.db")
}

// GetBackupDir returns the backup directory path
func (c *Config) GetBackupDir() string {
	if c.DataDir == "" {
		return "backups" // fallback
	}
	return filepath.Join(c.DataDir, "backups")
}

// GetEventLogPath returns the event log path
func (c *Config) GetEventLogPath() string {
	if c.DataDir == "" {
		return "events.ndjson" // fallback
	}
	return filepath.Join(c.DataDir, "events.ndjson")
}

// GetLogPath returns the log file path
func (c *Config) GetLogPath() string {
	if c.DataDir == "" {
		return "sentinel.log" // fallback
	}
	return filepath.Join(c.DataDir, "sentinel.log")
}

// GetKeysDir returns the keys directory path
func (c *Config) GetKeysDir() string {
	if c.DataDir == "" {
		return "keys" // fallback
	}
	return filepath.Join(c.DataDir, "keys")
}

// GetKeyPath returns the path for a specific key file
func (c *Config) GetKeyPath(keyname string) string {
	return filepath.Join(c.GetKeysDir(), keyname+".key")
}