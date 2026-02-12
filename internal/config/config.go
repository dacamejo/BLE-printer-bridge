package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
	} `toml:"server"`

	Auth struct {
		ApiKey string `toml:"api_key"`
	} `toml:"auth"`

	BLE struct {
		DeviceNameContains      string `toml:"device_name_contains"`
		PrinterAddress          string `toml:"printer_address"`
		ServiceUUID             string `toml:"service_uuid"`
		WriteCharacteristicUUID string `toml:"write_characteristic_uuid"`
		ChunkSize               int    `toml:"chunk_size"`
		WriteWithResponse       bool   `toml:"write_with_response"`
	} `toml:"ble"`

	Logging struct {
		FilePath       string `toml:"file_path"`
		ConsoleVerbose bool   `toml:"console_verbose"`
	} `toml:"logging"`

	CORS struct {
		AllowOrigins        string `toml:"allow_origins"`
		AllowOriginPatterns string `toml:"allow_origin_patterns"`
	} `toml:"cors"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	ApplyDefaults(&cfg)
	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func ApplyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 17800
	}
	if cfg.BLE.ChunkSize == 0 {
		cfg.BLE.ChunkSize = 180
	}
	if cfg.BLE.PrinterAddress == "" {
		cfg.BLE.PrinterAddress = "66:22:B6:5C:5C:3C"
	}
	if cfg.Logging.FilePath == "" {
		cfg.Logging.FilePath = "logs/app.log"
	}
	if cfg.CORS.AllowOrigins == "" {
		cfg.CORS.AllowOrigins = "https://integrated-pos.onrender.com/"
	}
	if cfg.CORS.AllowOriginPatterns == "" {
		cfg.CORS.AllowOriginPatterns = "https://integrated-pos-web-pr-*.onrender.com"
	}
}

func Save(path string, cfg *Config) error {
	ApplyDefaults(cfg)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := toml.NewEncoder(file)
	return encoder.Encode(cfg)
}

func applyEnvOverrides(cfg *Config) {
	if val := os.Getenv("BRIDGE_CORS_ALLOW_ORIGINS"); val != "" {
		cfg.CORS.AllowOrigins = val
	} else if val := os.Getenv("AGENT_CORS_ALLOW_ORIGINS"); val != "" {
		cfg.CORS.AllowOrigins = val
	}
	if val := os.Getenv("BRIDGE_CORS_ALLOW_ORIGIN_PATTERNS"); val != "" {
		cfg.CORS.AllowOriginPatterns = val
	} else if val := os.Getenv("AGENT_CORS_ALLOW_ORIGIN_PATTERNS"); val != "" {
		cfg.CORS.AllowOriginPatterns = val
	}
}
