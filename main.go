package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	srv := NewServer(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
