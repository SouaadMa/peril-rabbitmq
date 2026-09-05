package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPrefetch  = 10
	defaultHeartbeat = 5 * time.Second
)

type Config struct {
	AMQPURL           string
	MgmtURL           string
	ConsoleURL        string
	Prefetch          int
	WriteLogWait      time.Duration
	HeartbeatInterval time.Duration
}

func Load() (Config, error) {
	loadDotEnv(".env")

	cfg := Config{
		AMQPURL:           os.Getenv("AMQP_URL"),
		MgmtURL:           os.Getenv("RABBIT_MGMT_URL"),
		ConsoleURL:        os.Getenv("CONSOLE_URL"),
		Prefetch:          defaultPrefetch,
		HeartbeatInterval: defaultHeartbeat,
	}

	if cfg.AMQPURL == "" {
		return Config{}, errors.New("AMQP_URL is required")
	}

	if raw := os.Getenv("PREFETCH"); raw != "" {
		prefetch, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("PREFETCH %q is not a number: %w", raw, err)
		}
		if prefetch < 0 {
			return Config{}, fmt.Errorf("PREFETCH must not be negative, got %d", prefetch)
		}
		cfg.Prefetch = prefetch
	}

	if raw := os.Getenv("WRITE_LOG_WAIT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("WRITE_LOG_WAIT_SECONDS %q is not a number: %w", raw, err)
		}
		cfg.WriteLogWait = time.Duration(seconds) * time.Second
	}

	if raw := os.Getenv("HEARTBEAT_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("HEARTBEAT_SECONDS %q is not a number: %w", raw, err)
		}
		if seconds <= 0 {
			return Config{}, fmt.Errorf("HEARTBEAT_SECONDS must be positive, got %d", seconds)
		}
		cfg.HeartbeatInterval = time.Duration(seconds) * time.Second
	}

	return cfg, nil
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, value)
		}
	}
}
