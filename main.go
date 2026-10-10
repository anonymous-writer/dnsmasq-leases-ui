package main

// Application entry point: load configuration, create dependencies, and start HTTP.

import (
	"html/template"
	"log"
	"net/http"
)

func main() {
	cfg := loadConfig()
	known := newKnownMACs(cfg.KnownMACsPath)
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	handler := newHTTPHandler(cfg, known, tmpl)

	log.Printf("dnsmasq-leases-ui listening on %s:%s", cfg.Host, cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Host+":"+cfg.Port, handler))
}
