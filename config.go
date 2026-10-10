package main

// Load environment-based configuration and parse duration/integer options.

import (
	"os"
	"strconv"
	"time"
)

func envInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			return n
		}
	}
	return d
}

func envDuration(k string, d time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if x, e := time.ParseDuration(v); e == nil {
			return x
		}
		if n, e := strconv.ParseFloat(v, 64); e == nil {
			return time.Duration(n * float64(time.Second))
		}
	}
	return d
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// Config contains paths, server settings, and build metadata.
type Config struct {
	Host          string
	Port          string
	LeasePath     string
	HostsPath     string
	KnownMACsPath string
	AppVersion    string
	ReleaseDate   string
	RepoURL       string
}

// loadConfig reads runtime options from environment variables.
func loadConfig() Config {
	return Config{
		Host:          getenv("HOST", "0.0.0.0"),
		Port:          getenv("PORT", "5000"),
		LeasePath:     getenv("DNSMASQ_LEASES_FILE", "/var/lib/dnsmasq/dnsmasq.leases"),
		HostsPath:     getenv("DNSMASQ_HOSTS_FILE", "/var/lib/dnsmasq/dnsmasq.dhcphosts"),
		KnownMACsPath: getenv("KNOWN_MACS_FILE", "/data/known-macs.json"),
		AppVersion:    getenv("APP_VERSION", "dev"),
		ReleaseDate:   getenv("APP_RELEASE_DATE", ""),
		RepoURL:       getenv("REPO_URL", "https://github.com/anonymous-writer/dnsmasq-leases-ui"),
	}
}
