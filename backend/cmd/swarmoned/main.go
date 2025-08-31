package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/you/swarmone/internal/httpapi"
	"github.com/you/swarmone/internal/orch"
)

func main() {
	// Load .env from common locations (project root / backend / parent)
	_ = godotenv.Load(
		".env",
		"../.env",
		"../../.env",
		"./backend/.env",
	)

	cfg, keys, err := orch.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := httpapi.Serve(cfg, keys); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
