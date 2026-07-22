package config

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/gophish/gophish/logger"
)

// AdminServer represents the Admin server configuration details
type AdminServer struct {
	ListenURL            string   `json:"listen_url"`
	UseTLS               bool     `json:"use_tls"`
	CertPath             string   `json:"cert_path"`
	KeyPath              string   `json:"key_path"`
	CSRFKey              string   `json:"csrf_key"`
	AllowedInternalHosts []string `json:"allowed_internal_hosts"`
	TrustedOrigins       []string `json:"trusted_origins"`
}

// PhishServer represents the Phish server configuration details
type PhishServer struct {
	ListenURL string `json:"listen_url"`
	UseTLS    bool   `json:"use_tls"`
	CertPath  string `json:"cert_path"`
	KeyPath   string `json:"key_path"`
}

// Config represents the configuration information.
type Config struct {
	AdminConf      AdminServer `json:"admin_server"`
	PhishConf      PhishServer `json:"phish_server"`
	DBName         string      `json:"db_name"`
	DBPath         string      `json:"db_path"`
	DBSSLCaPath    string      `json:"db_sslca_path"`
	MigrationsPath string      `json:"migrations_prefix"`
	TestFlag       bool        `json:"test_flag"`
	ContactAddress string      `json:"contact_address"`
	Logging        *log.Config `json:"logging"`
	Theme          Theme       `json:"theme"`
}

// Theme represents the customizable branding options for the admin
// dashboard. Any field left blank in config.json falls back to the
// default Gophish look, set by applyThemeDefaults.
type Theme struct {
	PrimaryColor string `json:"primary_color"`
	SidebarColor string `json:"sidebar_color"`
	FontFamily   string `json:"font_family"`
	LogoURL      string `json:"logo_url"`
	BrandName    string `json:"brand_name"`
}

// defaultTheme matches the colors, font, and logo Gophish has always
// shipped with, so installs that don't set a "theme" block in
// config.json are unaffected.
var defaultTheme = Theme{
	PrimaryColor: "#222222",
	SidebarColor: "#283F50",
	FontFamily:   "'Source Sans Pro', Helvetica, Arial, sans-serif",
	LogoURL:      "/images/logo_inv_small.png",
	BrandName:    "gophish",
}

// applyThemeDefaults fills in any Theme fields left blank in config.json
// with Gophish's default branding.
func applyThemeDefaults(t Theme) Theme {
	if t.PrimaryColor == "" {
		t.PrimaryColor = defaultTheme.PrimaryColor
	}
	if t.SidebarColor == "" {
		t.SidebarColor = defaultTheme.SidebarColor
	}
	if t.FontFamily == "" {
		t.FontFamily = defaultTheme.FontFamily
	}
	if t.LogoURL == "" {
		t.LogoURL = defaultTheme.LogoURL
	}
	if t.BrandName == "" {
		t.BrandName = defaultTheme.BrandName
	}
	return t
}

// Version contains the current gophish version
var Version = ""

// CurrentTheme holds the active admin dashboard theme, set from the loaded
// Config in gophish.go so that it's accessible when building template
// parameters (mirroring the Version variable above).
var CurrentTheme = defaultTheme

// ServerName is the server type that is returned in the transparency response.
const ServerName = "gophish"

// LoadConfig loads the configuration from the specified filepath
func LoadConfig(filepath string) (*Config, error) {
	// Get the config file
	configFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(configFile, config)
	if err != nil {
		return nil, err
	}
	if config.Logging == nil {
		config.Logging = &log.Config{}
	}
	// Choosing the migrations directory based on the database used.
	config.MigrationsPath = config.MigrationsPath + config.DBName
	// Explicitly set the TestFlag to false to prevent config.json overrides
	config.TestFlag = false
	config.Theme = applyThemeDefaults(config.Theme)
	return config, nil
}
