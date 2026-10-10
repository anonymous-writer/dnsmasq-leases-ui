package main

// Probe devices and discover their HTTP/HTTPS management URLs.

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

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
	target := net.ParseIP(ip)
	if target == nil {
		return false
	}

	ch := make(chan bool, 1)
	go func() {
		addrs, err := net.LookupIP(host)
		if err != nil {
			ch <- false
			return
		}
		for _, a := range addrs {
			if a.Equal(target) {
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

func formatWebURL(scheme, ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if parsed.To4() == nil {
		return scheme + "://[" + ip + "]"
	}
	return scheme + "://" + ip
}

func checkWebUI(ip string, timeout time.Duration) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}

	// IPv6 literals must be enclosed in brackets when used in an HTTP URL.
	requestHost := ip
	if parsed.To4() == nil {
		requestHost = "[" + ip + "]"
	}

	for _, scheme := range []string{"http", "https"} {
		client := &http.Client{Timeout: timeout}
		if scheme == "https" {
			client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		}
		req, err := http.NewRequest(http.MethodGet, scheme+"://"+requestHost+"/", nil)
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
			return formatWebURL(scheme, ip)
		}
	}
	return ""
}

// pingReachable performs one bounded ICMP probe. If ping is unavailable or
// the probe cannot be performed, it returns false; callers should avoid
// interpreting that alone as definitive proof that a device is offline.

func pingReachable(ip string, timeout time.Duration) bool {
	if net.ParseIP(ip) == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// BusyBox and iputils support -c 1 and -W; the container image already
	// provides ping in the deployment environment.
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(max(1, int(timeout.Seconds()))), ip)
	return cmd.Run() == nil
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
				switch {
				case leases[n].WebURL != "":
					leases[n].Status = "online" // green: HTTP/HTTPS responded
				case pingReachable(leases[n].IPAddress, envDuration("PING_TIMEOUT", 1200*time.Millisecond)):
					leases[n].Status = "reachable" // yellow: host responds to ping, no web UI
				default:
					leases[n].Status = "offline" // red: no response to supported probes
				}
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
