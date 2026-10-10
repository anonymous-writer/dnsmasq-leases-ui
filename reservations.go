package main

// Parse dnsmasq DHCP host reservations and match them to leases.

import (
	"bufio"
	"net"
	"os"
	"strings"
)

func (r *Reservations) addLine(line string) {
	line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
	if line == "" {
		return
	}
	raw := strings.Split(line, ",")
	fields := make([]string, 0, len(raw))
	for _, f := range raw {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return
	}
	for _, f := range fields {
		c := strings.Trim(f, "[]")
		if ip := net.ParseIP(c); ip != nil {
			r.ips[ip.String()] = struct{}{}
			continue
		}
		if macRE.MatchString(c) {
			r.identifiers[normID(c)] = struct{}{}
		}
	}
	for i := len(fields) - 1; i >= 0; i-- {
		c := strings.Trim(strings.TrimSpace(fields[i]), "[]")
		l := strings.ToLower(c)
		if c == "" || net.ParseIP(c) != nil || macRE.MatchString(c) || leaseTimeRE.MatchString(c) || l == "ignore" {
			continue
		}
		if strings.Contains(c, "=") || strings.HasPrefix(l, "set:") || strings.HasPrefix(l, "tag:") || strings.HasPrefix(l, "id:") || strings.HasPrefix(l, "net:") || strings.HasPrefix(l, "bootfile=") || strings.HasPrefix(l, "pxe-service=") || strings.HasPrefix(l, "dhcp-option=") {
			continue
		}
		r.names[normName(c)] = struct{}{}
		break
	}
	first := strings.Trim(strings.TrimSpace(fields[0]), "[]")
	if first == "" || macRE.MatchString(first) || net.ParseIP(first) != nil {
		return
	}
	l := strings.ToLower(first)
	if strings.HasPrefix(l, "id:") {
		if v := strings.TrimSpace(first[3:]); v != "" && v != "*" {
			r.identifiers[normID(v)] = struct{}{}
		}
	} else if !strings.HasPrefix(l, "set:") && !strings.HasPrefix(l, "tag:") && !strings.HasPrefix(l, "net:") && !strings.HasPrefix(l, "bootfile=") {
		r.identifiers[normID(first)] = struct{}{}
	}
}

func readReservations(path string) *Reservations {
	r := newReservations()
	f, err := os.Open(path)
	if err != nil {
		return r
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		r.addLine(s.Text())
	}
	return r
}

func (r *Reservations) matches(identifier, ip, name, clientID string) bool {
	if p := net.ParseIP(ip); p != nil {
		if _, ok := r.ips[p.String()]; ok {
			return true
		}
	}
	if name != "" {
		if _, ok := r.names[normName(name)]; ok {
			return true
		}
	}
	if identifier != "" {
		if _, ok := r.identifiers[normID(identifier)]; ok {
			return true
		}
	}
	if clientID != "" {
		if _, ok := r.identifiers[normID(clientID)]; ok {
			return true
		}
	}
	return false
}
