// Package ntpcheck implements CON-11: query an NTP server for offset, stratum, delay and refid.
package ntpcheck

import (
	"net/http"
	"time"

	"github.com/beevik/ntp"
	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/rttstats"
)

// Tool is the NTP check.
type Tool struct {
	deps tools.Deps
	// Port overrides 123 (tests).
	Port int
}

func (*Tool) ID() string                                { return "net.ntp" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(10*time.Second)).Get("/ntp", t.handle)
}

// Result is the response payload.
type Result struct {
	Server           string  `json:"server"`
	Address          string  `json:"address"`
	Version          int     `json:"version"`
	Stratum          int     `json:"stratum"`
	ReferenceID      string  `json:"referenceId"`
	ReferenceTime    string  `json:"referenceTime"`
	ServerTime       string  `json:"serverTime"`
	OffsetMs         float64 `json:"offsetMs"`
	RTTMs            float64 `json:"rttMs"`
	PrecisionMs      float64 `json:"precisionMs"`
	RootDelayMs      float64 `json:"rootDelayMs"`
	RootDispersionMs float64 `json:"rootDispersionMs"`
	RootDistanceMs   float64 `json:"rootDistanceMs"`
	Leap             string  `json:"leap"`
	KissCode         string  `json:"kissCode"`
	Valid            bool    `json:"valid"`
	Problem          string  `json:"problem"`
}

var leapNames = map[ntp.LeapIndicator]string{
	ntp.LeapNoWarning: "none", ntp.LeapAddSecond: "insert second", ntp.LeapDelSecond: "delete second", ntp.LeapNotInSync: "not synchronized",
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("server", "", true, 300)
	family := q.Family()
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	tgt, err := t.deps.Target.Resolve(r.Context(), raw, target.Opts{Family: family, Field: "server"})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	port := t.Port
	if port == 0 {
		port = 123
	}
	resp, err := ntp.QueryWithOptions(tgt.IP.String(), ntp.QueryOptions{Timeout: 5 * time.Second, Port: port, Version: 4})
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "the NTP server did not answer", Err: err})
		return
	}
	res := Result{
		Server: raw, Address: tgt.IP.String(), Version: resp.Version, Stratum: int(resp.Stratum),
		ReferenceID: resp.ReferenceString(), ReferenceTime: resp.ReferenceTime.UTC().Format(time.RFC3339Nano),
		ServerTime: resp.Time.UTC().Format(time.RFC3339Nano),
		OffsetMs:   rttstats.Ms(resp.ClockOffset), RTTMs: rttstats.Ms(resp.RTT), PrecisionMs: rttstats.Ms(resp.Precision),
		RootDelayMs: rttstats.Ms(resp.RootDelay), RootDispersionMs: rttstats.Ms(resp.RootDispersion), RootDistanceMs: rttstats.Ms(resp.RootDistance),
		Leap: leapNames[resp.Leap], KissCode: resp.KissCode, Valid: true,
	}
	if err := resp.Validate(); err != nil {
		res.Valid, res.Problem = false, err.Error()
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: tgt.IP.String()}})
}
