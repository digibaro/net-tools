package geoip_test

import (
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/maxmind/mmdbwriter/mmdbtype"

	"github.com/digibaro/net-tools-docker/apps/api/internal/geoip"
	"github.com/digibaro/net-tools-docker/apps/api/internal/geoip/geoiptest"
)

func TestLookupAndReload(t *testing.T) {
	dir := t.TempDir()
	geoiptest.Write(t, dir, "GeoLite2-City.mmdb", "GeoLite2-City", map[string]mmdbtype.Map{
		"192.0.2.0/24": geoiptest.City("Amsterdam", "NL", "Netherlands", 52.37, 4.89),
	})
	path, explicit, err := geoip.Find(dir, "", geoip.CityFiles)
	if err != nil || explicit {
		t.Fatal(path, explicit, err)
	}
	db, err := geoip.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var c geoip.City
	prefix, found, err := db.Lookup(netip.MustParseAddr("192.0.2.7"), &c)
	if err != nil || !found || c.City.Names.Name() != "Amsterdam" || c.Country.ISOCode != "NL" || *c.Location.Latitude != 52.37 || prefix.String() != "192.0.2.0/24" {
		t.Fatalf("lookup = %+v %v %v %v", c, prefix, found, err)
	}
	if _, found, _ := db.Lookup(netip.MustParseAddr("198.51.100.1"), &c); found {
		t.Error("unexpected hit")
	}
	if typ, _ := db.Info(); typ != "GeoLite2-City" {
		t.Error(typ)
	}

	// Replace the file; the next lookup after CheckEvery picks it up.
	geoiptest.Write(t, dir, "GeoLite2-City.mmdb", "GeoLite2-City", map[string]mmdbtype.Map{
		"192.0.2.0/24": geoiptest.City("Rotterdam", "NL", "Netherlands", 51.92, 4.48),
	})
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	geoip.ForceCheck(db)
	c = geoip.City{}
	if _, _, _ = db.Lookup(netip.MustParseAddr("192.0.2.7"), &c); c.City.Names.Name() != "Rotterdam" {
		t.Errorf("after reload = %q", c.City.Names.Name())
	}

	if _, _, err := geoip.Find(t.TempDir(), "", geoip.ASNFiles); err == nil {
		t.Error("missing ASN db not reported")
	}
	if p, explicit, _ := geoip.Find("/data", "custom.mmdb", nil); p != "/data/custom.mmdb" || !explicit {
		t.Error(p)
	}
}
