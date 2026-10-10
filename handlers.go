package main

// HTTP handlers for the web interface and JSON API.

import (
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
)

// newHTTPHandler wires the page, static assets, and API endpoints together.
func newHTTPHandler(cfg Config, known *KnownMACs, tmpl *template.Template) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(cfg, tmpl))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/known-macs", knownMACsHandler(cfg, known))
	mux.HandleFunc("/leases", leasesHandler(cfg, known))
	return mux
}

func homeHandler(cfg Config, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := PageData{cfg.AppVersion, cfg.ReleaseDate, cfg.RepoURL}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
		}
	}
}

// knownMACsHandler supports individual mark/unmark, reset, and bulk-marking
// all currently detected DHCP reservations.
func knownMACsHandler(cfg Config, known *KnownMACs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			MAC    string `json:"mac"`
			Action string `json:"action"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body) != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		var err error
		markedCount := 0
		switch body.Action {
		case "", "mark":
			err = known.mark(body.MAC)
		case "unmark":
			err = known.unmark(body.MAC)
		case "reset":
			err = known.reset()
		case "mark-reservations":
			var leases []LeaseEntry
			leases, err = readLeaseEntries(cfg.LeasePath, cfg.HostsPath)
			if err == nil {
				macs := make([]string, 0, len(leases))
				for _, lease := range leases {
					mac := strings.TrimSpace(lease.MACAddress)
					if lease.StaticIP && macRE.MatchString(mac) {
						macs = append(macs, mac)
					}
				}
				markedCount, err = known.markMany(macs)
			}
		default:
			http.Error(w, "invalid action", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "cannot save known MACs (check KNOWN_MACS_FILE permissions)", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if body.Action == "mark-reservations" {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": markedCount})
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}

func leasesHandler(cfg Config, known *KnownMACs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		leases, err := readLeases(cfg.LeasePath, cfg.HostsPath)
		if err != nil {
			log.Printf("cannot read %s: %v", cfg.LeasePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"leases file unavailable"}`))
			return
		}
		for i := range leases {
			mac := strings.TrimSpace(leases[i].MACAddress)
			ip := net.ParseIP(leases[i].IPAddress)
			// IPv6 entries without a valid MAC stay visible but are not labelled new.
			if ip != nil && ip.To4() == nil && !macRE.MatchString(mac) {
				leases[i].IsNew = false
				continue
			}
			leases[i].IsNew = !known.has(mac)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"leases": leases})
	}
}
