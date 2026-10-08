package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LeaseEntry struct {
	StaticIP   bool   `json:"staticIP"`
	LeaseTime  string `json:"leasetime"`
	MACAddress string `json:"macAddress"`
	IPAddress  string `json:"ipAddress"`
	Name       string `json:"name"`
	WebURL     string `json:"webUrl"`
	WebHostURL string `json:"webHostUrl"`
}

type Reservations struct{ ips, names, identifiers map[string]struct{} }

func newReservations() *Reservations {
	return &Reservations{map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}}
}

var macRE = regexp.MustCompile(`^(?:[0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
var leaseTimeRE = regexp.MustCompile(`^(?:\d+[smhdw]?|infinite)$`)

func normName(s string) string { return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), ".")) }
func normID(s string) string   { return strings.ToLower(strings.TrimSpace(s)) }

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
	return LeaseEntry{StaticIP: p[0] == "0" || r.matches(p[1], p[2], p[3], p[4]), LeaseTime: end, MACAddress: strings.ToUpper(p[1]), IPAddress: p[2], Name: p[3]}, true
}
func readLeases(leasePath, hostsPath string) ([]LeaseEntry, error) {
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
	addWebURLs(leases)
	return leases, nil
}
func searchDomains() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		p := strings.Fields(s.Text())
		if len(p) < 2 || (p[0] != "search" && p[0] != "domain") {
			continue
		}
		for _, v := range p[1:] {
			v = strings.Trim(v, ".")
			if v != "" && !contains(out, v) {
				out = append(out, v)
			}
		}
	}
	return out
}
func contains(a []string, v string) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}
func hostnameCandidates(name string) []string {
	if name == "" || name == "*" {
		return nil
	}
	name = strings.TrimSuffix(name, ".")
	if strings.Contains(name, ".") {
		return []string{name}
	}
	var out []string
	for _, s := range searchDomains() {
		x := name + "." + s
		if !contains(out, x) {
			out = append(out, x)
		}
	}
	x := name + ".local"
	if !contains(out, x) {
		out = append(out, x)
	}
	return out
}
func resolvesToIP(host, ip string, timeout time.Duration) bool {
	ch := make(chan bool, 1)
	go func() {
		addrs, err := net.LookupIP(host)
		if err != nil {
			ch <- false
			return
		}
		for _, a := range addrs {
			if a.String() == ip {
				ch <- true
				return
			}
		}
		ch <- false
	}()
	select {
	case ok := <-ch:
		return ok
	case <-time.After(timeout):
		return false
	}
}
func checkWebHostname(ip, name, webURL string, timeout time.Duration) string {
	if webURL == "" {
		return ""
	}
	scheme := "http"
	if strings.HasPrefix(webURL, "https://") {
		scheme = "https"
	}
	for _, h := range hostnameCandidates(name) {
		if resolvesToIP(h, ip, timeout) {
			return scheme + "://" + h
		}
	}
	return ""
}
func checkWebUI(ip string, timeout time.Duration) string {
	if net.ParseIP(ip) == nil {
		return ""
	}
	for _, scheme := range []string{"http", "https"} {
		client := &http.Client{Timeout: timeout}
		if scheme == "https" {
			client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		}
		req, err := http.NewRequest(http.MethodGet, scheme+"://"+ip+"/", nil)
		if err != nil {
			continue
		}
		req.Header.Set("Connection", "close")
		req.Header.Set("User-Agent", "dnsmasq-leases-ui")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 100 && resp.StatusCode <= 599 {
			return scheme + "://" + ip
		}
	}
	return ""
}
func addWebURLs(leases []LeaseEntry) {
	if len(leases) == 0 {
		return
	}
	timeout := envDuration("WEB_UI_TIMEOUT", 500*time.Millisecond)
	workers := envInt("WEB_UI_MAX_WORKERS", 16)
	if workers < 1 {
		workers = 1
	}
	if workers > len(leases) {
		workers = len(leases)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range jobs {
				leases[n].WebURL = checkWebUI(leases[n].IPAddress, timeout)
			}
		}()
	}
	for i := range leases {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for i := range leases {
		leases[i].WebHostURL = checkWebHostname(leases[i].IPAddress, leases[i].Name, leases[i].WebURL, timeout)
	}
}
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

type PageData struct{ Version, ReleaseDate, RepoURL string }

func main() {
	leasePath := getenv("DNSMASQ_LEASES_FILE", "/var/lib/dnsmasq/dnsmasq.leases")
	hostsPath := getenv("DNSMASQ_HOSTS_FILE", "/var/lib/dnsmasq/dnsmasq.dhcphosts")
	port := getenv("PORT", "5000")
	host := getenv("HOST", "0.0.0.0")
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := PageData{getenv("APP_VERSION", "dev"), getenv("APP_RELEASE_DATE", ""), getenv("REPO_URL", "https://github.com/fschlag/dnsmasq-leases-ui")}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "template error", 500)
		}
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/leases", func(w http.ResponseWriter, r *http.Request) {
		leases, err := readLeases(leasePath, hostsPath)
		if err != nil {
			log.Printf("cannot read %s: %v", leasePath, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"leases file unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"leases": leases})
	})
	log.Printf("dnsmasq-leases-ui listening on %s:%s", host, port)
	log.Fatal(http.ListenAndServe(host+":"+port, mux))
}
