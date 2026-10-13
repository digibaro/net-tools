package geoip

import "time"

// ForceCheck makes the next lookup re-stat the file.
func ForceCheck(d *DB) {
	d.mu.Lock()
	d.checked = time.Time{}
	d.mu.Unlock()
}
