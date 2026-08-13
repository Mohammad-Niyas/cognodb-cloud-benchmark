package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type DBConfig struct {
	Name     string
	URI      string
	User     string
	Password string
	Database string
}

type Config struct {
	CognoDB  DBConfig
	Neo4j    DBConfig
	Memgraph DBConfig
	FalkorDB DBConfig
	ArangoDB DBConfig

	BenchmarkIterations int
	WarmupIterations    int
	ConcurrencyWorkers  int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		CognoDB: DBConfig{
			Name:     "CognoDB Cloud",
			URI:      os.Getenv("COGNODB_URI"),
			User:     os.Getenv("COGNODB_USER"),
			Password: os.Getenv("COGNODB_PASSWORD"),
		},
		Neo4j: DBConfig{
			Name:     "Neo4j AuraDB",
			URI:      os.Getenv("NEO4J_URI"),
			User:     os.Getenv("NEO4J_USER"),
			Password: os.Getenv("NEO4J_PASSWORD"),
		},
		Memgraph: DBConfig{
			Name:     "Memgraph Cloud",
			URI:      os.Getenv("MEMGRAPH_URI"),
			User:     os.Getenv("MEMGRAPH_USER"),
			Password: os.Getenv("MEMGRAPH_PASSWORD"),
		},
		FalkorDB: DBConfig{
			Name:     "FalkorDB Cloud",
			URI:      os.Getenv("FALKORDB_URI"),
			User:     os.Getenv("FALKORDB_USER"),
			Password: os.Getenv("FALKORDB_PASSWORD"),
		},
		ArangoDB: DBConfig{
			Name:     "ArangoDB Remote VM",
			URI:      os.Getenv("ARANGO_URI"),
			User:     os.Getenv("ARANGO_USER"),
			Password: os.Getenv("ARANGO_PASSWORD"),
			Database: os.Getenv("ARANGO_DB"),
		},
		BenchmarkIterations: getEnvInt("BENCHMARK_ITERATIONS", 100),
		WarmupIterations:    getEnvInt("WARMUP_ITERATIONS", 10),
		ConcurrencyWorkers:  getEnvInt("CONCURRENCY_WORKERS", 10),
	}

	if cfg.CognoDB.URI == "" && cfg.Neo4j.URI == "" && cfg.Memgraph.URI == "" && cfg.FalkorDB.URI == "" && cfg.ArangoDB.URI == "" {
		return nil, fmt.Errorf("no database credentials found in environment or .env file")
	}

	return cfg, nil
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}