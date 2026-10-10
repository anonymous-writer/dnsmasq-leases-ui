package main

// Application entry point: load configuration, create dependencies, and start HTTP.
//
// Application messages are written through logging.go so LOG_LEVEL controls
// how much detail is printed to the container log.

import (
	"html/template"
	"net/http"
	"os"
)

func main() {
	cfg := loadConfig()
	known := newKnownMACs(cfg.KnownMACsPath)

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		LogFatalf("cannot parse template templates/index.html: %v", err)
	}

	handler := newHTTPHandler(cfg, known, tmpl)
	address := cfg.Host + ":" + cfg.Port
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "dev"
	}

	LogInfof("dnsmasq-leases-ui starting version=%s listen=%s", version, address)
	if err := http.ListenAndServe(address, handler); err != nil {
		LogFatalf("HTTP server stopped: %v", err)
	}
}
