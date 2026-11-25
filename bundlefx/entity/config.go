package entity

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/addpur/custome-speedtest/common"
	"github.com/addpur/custome-speedtest/speedtest/domain"
	"go.uber.org/fx"
)

func NewConfig() domain.Config {
	keywordsStr := getEnv("KEYWORDS", "")
	var keywords []string
	if keywordsStr != "" {
		for _, kw := range strings.Split(keywordsStr, ",") {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				keywords = append(keywords, kw)
			}
		}
	}

	serversStr := getEnv("SERVERS", "")
	var servers []string
	if serversStr != "" {
		for _, s := range strings.Split(serversStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				servers = append(servers, s)
			}
		}
	}

	durationSec := getEnvInt("DEFAULT_DURATION", common.DefaultDuration)

	return domain.Config{
		ListenAddr:     getEnv("LISTEN_ADDR", common.DefaultListenAddr),
		BufferSize:     getEnvInt("BUFFER_SIZE", common.DefaultBufferSize),
		Duration:       time.Duration(durationSec) * time.Second,
		Parallel:       getEnvInt("DEFAULT_PARALLEL", common.DefaultParallel),
		Overhead:       getEnvFloat("DEFAULT_OVERHEAD", common.DefaultOverhead),
		Keywords:       keywords,
		Servers:        servers,
		AutoRun:        getEnvBool("AUTO_RUN", false),
		AutoAfter:      time.Duration(getEnvInt("AUTO_AFTER", 0)) * time.Second,
		StressDuration: time.Duration(getEnvInt("STRESS_DURATION", 10)) * time.Second,
		Mode:           getEnv("MODE", "standalone"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		_, _ = fmt.Sscanf(v, "%d", &i)
		if i > 0 {
			return i
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		_, _ = fmt.Sscanf(v, "%f", &f)
		if f > 0 {
			return f
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(strings.TrimSpace(v))
		return v == "true" || v == "1" || v == "yes"
	}
	return fallback
}

func RegisterSignals(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}
