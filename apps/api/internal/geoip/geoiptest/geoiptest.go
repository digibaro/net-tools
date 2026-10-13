// Package geoiptest writes small mmdb fixtures for tests.
package geoiptest

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// Write creates dir/name with the given network → record entries.
func Write(t *testing.T, dir, name, dbType string, entries map[string]mmdbtype.Map) string {
	t.Helper()
	tree, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: dbType, IncludeReservedNetworks: true, RecordSize: 28})
	if err != nil {
		t.Fatal(err)
	}
	for cidr, rec := range entries {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatal(err)
		}
		if err := tree.Insert(n, rec); err != nil {
			t.Fatal(err)
		}
	}
	p := filepath.Join(dir, name)
	f, err := os.Create(p) //nolint:gosec // test path
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := tree.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	return p
}

// City is a GeoLite2-City-shaped record.
func City(city, countryISO, country string, lat, lon float64) mmdbtype.Map {
	return mmdbtype.Map{
		"city":      mmdbtype.Map{"names": mmdbtype.Map{"en": mmdbtype.String(city)}},
		"country":   mmdbtype.Map{"iso_code": mmdbtype.String(countryISO), "names": mmdbtype.Map{"en": mmdbtype.String(country)}},
		"continent": mmdbtype.Map{"code": mmdbtype.String("EU"), "names": mmdbtype.Map{"en": mmdbtype.String("Europe")}},
		"location":  mmdbtype.Map{"latitude": mmdbtype.Float64(lat), "longitude": mmdbtype.Float64(lon), "accuracy_radius": mmdbtype.Uint16(20), "time_zone": mmdbtype.String("Europe/Amsterdam")},
	}
}

// ASN is a GeoLite2-ASN-shaped record.
func ASN(n uint32, org string) mmdbtype.Map {
	return mmdbtype.Map{"autonomous_system_number": mmdbtype.Uint32(n), "autonomous_system_organization": mmdbtype.String(org)}
}
