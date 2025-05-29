package config

import (
	"flag"
	"strings"
)

type Config struct {
	Port      int
	TargetURL string
	DBPath    string
}

func ParseFlags() Config {
	port := flag.Int("port", 8080, "Port to listen on")
	targetURL := flag.String("url", "https://router.requesty.ai/v1", "URL of the target API")
	dbPath := flag.String("db", "ai-gateway.db", "Path to SQLite database file")

	flag.Parse()

	return Config{
		Port:      *port,
		TargetURL: strings.TrimSuffix(*targetURL, "/"),
		DBPath:    *dbPath,
	}
}