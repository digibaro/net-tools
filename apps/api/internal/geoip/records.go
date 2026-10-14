package geoip

// Names holds localized names keyed by language.
type Names map[string]string

// City is the GeoLite2/DB-IP City record subset used here.
type City struct {
	City struct {
		Names Names `maxminddb:"names"`
	} `maxminddb:"city"`
	Continent struct {
		Code  string `maxminddb:"code"`
		Names Names  `maxminddb:"names"`
	} `maxminddb:"continent"`
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
		Names   Names  `maxminddb:"names"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		ISOCode string `maxminddb:"iso_code"`
		Names   Names  `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Postal struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"postal"`
	Location struct {
		Latitude       *float64 `maxminddb:"latitude"`
		Longitude      *float64 `maxminddb:"longitude"`
		AccuracyRadius uint16   `maxminddb:"accuracy_radius"`
		TimeZone       string   `maxminddb:"time_zone"`
	} `maxminddb:"location"`
}

// ASN is the GeoLite2/DB-IP ASN record.
type ASN struct {
	Number uint32 `maxminddb:"autonomous_system_number"`
	Org    string `maxminddb:"autonomous_system_organization"`
}

// Name picks English, falling back to any language.
func (n Names) Name() string {
	if s := n["en"]; s != "" {
		return s
	}
	for _, s := range n {
		return s
	}
	return ""
}
