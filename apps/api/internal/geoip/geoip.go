// Package geoip reads MaxMind-format databases (GeoLite2, DB-IP Lite) from the data directory
// and reloads them when the file changes (docs/API_SPEC.md §7).
package geoip

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"
)

// CheckEvery bounds how often the file's modification time is checked.
const CheckEvery = 30 * time.Second

// DB is a hot-reloading mmdb reader.
type DB struct {
	path    string
	mu      sync.RWMutex
	r       *maxminddb.Reader
	mod     time.Time
	checked time.Time
}

// Open loads path.
func Open(path string) (*DB, error) {
	d := &DB{path: path}
	if err := d.load(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) load() error {
	st, err := os.Stat(d.path)
	if err != nil {
		return fmt.Errorf("geoip: %w", err)
	}
	r, err := maxminddb.Open(d.path)
	if err != nil {
		return fmt.Errorf("geoip: open %s: %w", d.path, err)
	}
	old := d.r
	d.r, d.mod, d.checked = r, st.ModTime(), time.Now()
	if old != nil {
		_ = old.Close() // readers hold d.mu.RLock while using it
	}
	return nil
}

func (d *DB) maybeReload() {
	d.mu.RLock()
	due := time.Since(d.checked) >= CheckEvery
	d.mu.RUnlock()
	if !due {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if time.Since(d.checked) < CheckEvery {
		return
	}
	d.checked = time.Now()
	if st, err := os.Stat(d.path); err == nil && !st.ModTime().Equal(d.mod) {
		_ = d.load() // keep serving the old file if the new one is broken
	}
}

// Lookup decodes the record for ip into v; found is false when the database has no entry.
func (d *DB) Lookup(ip netip.Addr, v any) (prefix netip.Prefix, found bool, err error) {
	d.maybeReload()
	d.mu.RLock()
	defer d.mu.RUnlock()
	res := d.r.Lookup(ip.Unmap())
	if err := res.Err(); err != nil {
		return netip.Prefix{}, false, fmt.Errorf("geoip: %w", err)
	}
	if !res.Found() {
		return netip.Prefix{}, false, nil
	}
	if err := res.Decode(v); err != nil {
		return netip.Prefix{}, false, fmt.Errorf("geoip: decode: %w", err)
	}
	return res.Prefix(), true, nil
}

// Info describes the loaded database.
func (d *DB) Info() (dbType string, built time.Time) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	m := d.r.Metadata
	return m.DatabaseType, time.Unix(int64(m.BuildEpoch), 0).UTC() //nolint:gosec // epoch fits
}

// Path is the database file.
func (d *DB) Path() string { return d.path }

// Default file names looked for in the data directory when no path is configured.
var (
	CityFiles = []string{"GeoLite2-City.mmdb", "dbip-city-lite.mmdb", "GeoIP2-City.mmdb"}
	ASNFiles  = []string{"GeoLite2-ASN.mmdb", "dbip-asn-lite.mmdb", "GeoIP2-ISP.mmdb"}
)

// ErrNotFound means no database is configured or present.
var ErrNotFound = errors.New("geoip: no database")

// Find returns the database path: the configured one (relative to dataDir) or the first
// default name present in dataDir. explicit reports whether it was configured.
func Find(dataDir, configured string, defaults []string) (path string, explicit bool, err error) {
	if configured != "" {
		if !filepath.IsAbs(configured) {
			configured = filepath.Join(dataDir, configured)
		}
		return configured, true, nil
	}
	for _, name := range defaults {
		p := filepath.Join(dataDir, name)
		if _, err := os.Stat(p); err == nil {
			return p, false, nil
		}
	}
	return "", false, ErrNotFound
}

// ReadyCheck fails when a database path is configured explicitly but the file is missing.
// Auto-detected databases are optional and never block readiness.
func ReadyCheck(dataDir string, configured ...string) error {
	for _, c := range configured {
		if c == "" {
			continue
		}
		p, _, _ := Find(dataDir, c, nil)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("geoip: %w", err)
		}
	}
	return nil
}
