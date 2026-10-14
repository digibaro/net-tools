package speedtest

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func TestSpeedtest(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Env: map[string]string{"SPEEDTEST_ENABLED": "true"}})
	r := env.Get("/api/v1/speedtest/download?bytes=" + "3145728")
	body := r.Raw.Body.Bytes()
	if r.Status != 200 || len(body) != 3<<20 || r.Raw.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("download: %d %d", r.Status, len(body))
	}
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write(body[:chunk])
	_ = zw.Close()
	if gz.Len() < chunk*9/10 {
		t.Errorf("download data compresses to %d bytes: not random", gz.Len())
	}
	env.Get("/api/v1/speedtest/download?bytes=999999999").ExpectError(t, "INVALID_INPUT")

	// Uploads larger than the global 1 MiB body cap are accepted up to MaxUpload…
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/speedtest/upload", bytes.NewReader(make([]byte, 5<<20)))
	var up UploadResult
	env.Do(req).Must(t).Decode(t, &up)
	if up.Bytes != 5<<20 {
		t.Errorf("upload = %+v", up)
	}
	// …and cut beyond it.
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/speedtest/upload", bytes.NewReader(make([]byte, MaxUpload+1)))
	env.Do(req).ExpectError(t, "INVALID_INPUT")
	if p := env.Get("/api/v1/speedtest/ping"); p.Status != 200 || !strings.Contains(p.Raw.Body.String(), "pong") {
		t.Errorf("ping = %d", p.Status)
	}
}

func TestSpeedtestDisabledInPublicProfile(t *testing.T) {
	ok, reason := (&Tool{}).Enabled(tools.Deps{Config: tooltest.Config(t, map[string]string{"NETTOOLS_PROFILE": "public"})})
	if ok || reason != tools.ReasonDisabledByAdmin {
		t.Errorf("enabled=%v reason=%s", ok, reason)
	}
}
