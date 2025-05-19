package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	Address     string
	StaticDir   string
	EnableRss   bool
	EnableDebug bool
	RedisHost   string
	RedisPort   string
	RedisConns  int
	RedisMaxConns int
	RedisPassword string
}

var cfg Config

func initConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	cfg = Config{
		Port:         getEnv("PORT", "8080"),
		Address:      getEnv("ADDRESS", "0.0.0.0"),
		StaticDir:    getEnv("STATIC_DIR", "./static"),
		EnableRss:    getEnvAsBool("ENABLE_RSS", true),
		EnableDebug:  getEnvAsBool("ENABLE_DEBUG", false),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		RedisConns:   getEnvAsInt("REDIS_CONNS", 10),
		RedisMaxConns: getEnvAsInt("REDIS_MAX_CONNS", 100),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsInt(name string, defaultVal int) int {
	valStr := getEnv(name, "")
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
