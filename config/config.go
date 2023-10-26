package config

import (
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var (
	DefaultPath string
	configDir   string
	Debug       = log.New(os.Stderr, "", 0)
)

func init() {
	var err error
	configDir, err = os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to get config directory: %v", err)
	}
	DefaultPath = path.Join(configDir, "gcal-notify", "config.toml")
}

var Cfg = struct {
	ClientSecretPath     string
	TokenPath            string
	CalendarID           string
	PollInterval         Duration
	LookaheadInterval    Duration
	LocationPollInterval Duration
	SlackTokenFile       string
	Debug                bool
}{
	PollInterval:         Duration{3 * time.Minute},
	LookaheadInterval:    Duration{24 * time.Hour},
	LocationPollInterval: Duration{15 * time.Minute},
}

func Parse(configFilePath string) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		log.Printf("Failed to open configuration file, using defaults: %v", err)
	} else {
		defer configFile.Close()
		dec := toml.NewDecoder(configFile)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&Cfg); err != nil {
			log.Fatalf("Failed to decode configuration file: %v", err)
		}
	}

	if Cfg.ClientSecretPath == "" {
		Cfg.ClientSecretPath = path.Join(configDir, "gcal-notify", "client-secret.json")
	}
	if Cfg.TokenPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatalf("Failed to get cache directory: %v", err)
		}
		Cfg.TokenPath = path.Join(cacheDir, "gcal-notify", "token.json")
	}
	if Cfg.CalendarID == "" {
		log.Fatal("No calendar ID configured")
	}
	if Cfg.SlackTokenFile == "" {
		Cfg.SlackTokenFile = path.Join(configDir, "gcal-notify", "slack-token")
	}

	if !Cfg.Debug {
		Debug.SetOutput(io.Discard)
	}
}

type Duration struct{ D time.Duration }

func (d *Duration) UnmarshalText(data []byte) (err error) {
	d.D, err = time.ParseDuration(string(data))
	return
}
