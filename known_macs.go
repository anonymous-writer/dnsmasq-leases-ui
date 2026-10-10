package main

// Persist devices explicitly marked as known.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type KnownMACs struct {
	mu   sync.Mutex
	path string
	macs map[string]bool
}

func newKnownMACs(path string) *KnownMACs {
	k := &KnownMACs{path: path, macs: map[string]bool{}}
	data, err := os.ReadFile(path)
	if err == nil {
		var values []string
		if json.Unmarshal(data, &values) == nil {
			for _, m := range values {
				k.macs[strings.ToUpper(strings.TrimSpace(m))] = true
			}
		}
	}
	return k
}

func (k *KnownMACs) has(mac string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.macs[strings.ToUpper(strings.TrimSpace(mac))]
}

func (k *KnownMACs) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(k.path), 0755); err != nil {
		return err
	}
	values := make([]string, 0, len(k.macs))
	for m := range k.macs {
		values = append(values, m)
	}
	sort.Strings(values)
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(k.path, data, 0644)
}

func (k *KnownMACs) mark(mac string) error {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if !macRE.MatchString(mac) {
		return http.ErrNotSupported
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	wasKnown := k.macs[mac]
	k.macs[mac] = true
	if err := k.saveLocked(); err != nil {
		if !wasKnown {
			delete(k.macs, mac)
		}
		return err
	}
	return nil
}

// markMany remembers each unique valid MAC in one persistent write.
// It returns the number of unique valid MAC addresses supplied.

func (k *KnownMACs) markMany(macs []string) (int, error) {
	unique := make(map[string]struct{}, len(macs))
	for _, mac := range macs {
		mac = strings.ToUpper(strings.TrimSpace(mac))
		if macRE.MatchString(mac) {
			unique[mac] = struct{}{}
		}
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	added := make([]string, 0, len(unique))
	for mac := range unique {
		if !k.macs[mac] {
			k.macs[mac] = true
			added = append(added, mac)
		}
	}
	if len(added) == 0 {
		return len(unique), nil
	}
	if err := k.saveLocked(); err != nil {
		for _, mac := range added {
			delete(k.macs, mac)
		}
		return 0, err
	}
	return len(unique), nil
}

func (k *KnownMACs) unmark(mac string) error {
	mac = strings.ToUpper(strings.TrimSpace(mac))
	if !macRE.MatchString(mac) {
		return http.ErrNotSupported
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	wasKnown := k.macs[mac]
	delete(k.macs, mac)
	if err := k.saveLocked(); err != nil {
		if wasKnown {
			k.macs[mac] = true
		}
		return err
	}
	return nil
}

func (k *KnownMACs) reset() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	previous := k.macs
	k.macs = map[string]bool{}
	if err := k.saveLocked(); err != nil {
		k.macs = previous
		return err
	}
	return nil
}
