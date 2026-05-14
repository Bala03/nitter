package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	Address      string
	Port         int
	UseHTTPS     bool
	HTTPMaxConns int
	Title        string
	Hostname     string
	StaticDir    string

	// Cache
	ListCacheTime int
	RSSCacheTime  int
	RedisHost     string
	RedisPort     int
	RedisConns    int
	RedisMaxConns int
	RedisPassword string

	// Config
	HMACKey     string
	Base64Media bool
	MinTokens   int
	EnableRSS   bool
	EnableDebug bool
	Proxy       string
	ProxyAuth   string
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{
		Address:       "0.0.0.0",
		Port:          8080,
		UseHTTPS:      false,
		HTTPMaxConns:  100,
		Title:         "Nitter",
		Hostname:      "nitter.net",
		StaticDir:     "./public",
		ListCacheTime: 120,
		RSSCacheTime:  10,
		RedisHost:     "localhost",
		RedisPort:     6379,
		RedisConns:    20,
		RedisMaxConns: 30,
		RedisPassword: "",
		HMACKey:       "secretkey",
		Base64Media:   false,
		MinTokens:     10,
		EnableRSS:     true,
		EnableDebug:   false,
		Proxy:         "",
		ProxyAuth:     "",
	}

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Remove inline comments
		if idx := strings.Index(val, " #"); idx != -1 {
			val = strings.TrimSpace(val[:idx])
		}
		// Remove quotes
		val = strings.Trim(val, "\"")

		switch section {
		case "server":
			switch key {
			case "address":
				cfg.Address = val
			case "port":
				cfg.Port = parseInt(val, 8080)
			case "https":
				cfg.UseHTTPS = parseBool(val)
			case "httpMaxConnections":
				cfg.HTTPMaxConns = parseInt(val, 100)
			case "title":
				cfg.Title = val
			case "hostname":
				cfg.Hostname = val
			case "staticDir":
				cfg.StaticDir = val
			}
		case "cache":
			switch key {
			case "listMinutes":
				cfg.ListCacheTime = parseInt(val, 120)
			case "rssMinutes":
				cfg.RSSCacheTime = parseInt(val, 10)
			case "redisHost":
				cfg.RedisHost = val
			case "redisPort":
				cfg.RedisPort = parseInt(val, 6379)
			case "redisConnections":
				cfg.RedisConns = parseInt(val, 20)
			case "redisMaxConnections":
				cfg.RedisMaxConns = parseInt(val, 30)
			case "redisPassword":
				cfg.RedisPassword = val
			}
		case "config":
			switch key {
			case "hmacKey":
				cfg.HMACKey = val
			case "base64Media":
				cfg.Base64Media = parseBool(val)
			case "tokenCount":
				cfg.MinTokens = parseInt(val, 10)
			case "enableRSS":
				cfg.EnableRSS = parseBool(val)
			case "enableDebug":
				cfg.EnableDebug = parseBool(val)
			case "proxy":
				cfg.Proxy = val
			case "proxyAuth":
				cfg.ProxyAuth = val
			}
		}
	}

	return cfg, scanner.Err()
}

func parseInt(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1"
}

func (c *Config) BaseURL() string {
	scheme := "http"
	if c.UseHTTPS {
		scheme = "https"
	}
	return scheme + "://" + c.Hostname
}
