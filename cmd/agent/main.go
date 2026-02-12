package main

import (
	"log"

	"ble-printer-bridge/internal/config"
	"ble-printer-bridge/internal/httpapi"
	"ble-printer-bridge/internal/logging"
)

func main() {
	configPath := "config.toml"
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger, err := logging.New(cfg.Logging.FilePath, cfg.Logging.ConsoleVerbose)
	if err != nil {
		log.Fatalf("logging error: %v", err)
	}
	defer logger.Close()

	srv := httpapi.NewServer(cfg, configPath, logger)
	if err := srv.Run(); err != nil {
		logger.Error("server stopped: %v", err)
		log.Fatal(err)
	}
}
