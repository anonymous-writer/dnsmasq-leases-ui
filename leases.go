package main

// Read and parse entries from the dnsmasq lease file.

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

func formatLeaseEnd(v string) (string, bool) {
	if v == "0" {
		return "Never", true
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return "", false
	}
	return time.Unix(n, 0).Local().Format("2006-01-02 15:04:05"), true
}

func parseLeaseLine(line string, r *Reservations) (LeaseEntry, bool) {
	p := strings.Fields(line)
	if len(p) != 5 {
		return LeaseEntry{}, false
	}
	end, ok := formatLeaseEnd(p[0])
	if !ok {
		return LeaseEntry{}, false
	}
	expiry, _ := strconv.ParseInt(p[0], 10, 64)
	return LeaseEntry{StaticIP: p[0] == "0" || r.matches(p[1], p[2], p[3], p[4]), LeaseTime: end, LeaseExpiry: expiry, MACAddress: strings.ToUpper(p[1]), IPAddress: p[2], Name: p[3]}, true
}

func readLeaseEntries(leasePath, hostsPath string) ([]LeaseEntry, error) {
	f, err := os.Open(leasePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := readReservations(hostsPath)
	var leases []LeaseEntry
	s := bufio.NewScanner(f)
	for s.Scan() {
		if x, ok := parseLeaseLine(s.Text(), r); ok {
			leases = append(leases, x)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return leases, nil
}

func readLeases(leasePath, hostsPath string) ([]LeaseEntry, error) {
	leases, err := readLeaseEntries(leasePath, hostsPath)
	if err != nil {
		return nil, err
	}
	addWebURLs(leases)
	return leases, nil
}
