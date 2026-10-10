package main

// Shared data structures and normalization helpers.

import (
	"regexp"
	"strings"
)

type LeaseEntry struct {
	StaticIP    bool   `json:"staticIP"`
	LeaseTime   string `json:"leasetime"`
	LeaseExpiry int64  `json:"leaseExpiry"`
	MACAddress  string `json:"macAddress"`
	IPAddress   string `json:"ipAddress"`
	Name        string `json:"name"`
	WebURL      string `json:"webUrl"`
	WebHostURL  string `json:"webHostUrl"`
	Status      string `json:"status"`
	IsNew       bool   `json:"isNew"`
}

type Reservations struct{ ips, names, identifiers map[string]struct{} }

func newReservations() *Reservations {
	return &Reservations{map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}}
}

var macRE = regexp.MustCompile(`^(?:[0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)

var leaseTimeRE = regexp.MustCompile(`^(?:\d+[smhdw]?|infinite)$`)

func normName(s string) string { return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), ".")) }

func normID(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func contains(a []string, v string) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

type PageData struct{ Version, ReleaseDate, RepoURL string }
