package providers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/providers"
	"github.com/digibaro/net-tools/apps/api/internal/target"
)

func TestGetJSONCachesAndMapsErrors(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/ok":
			if r.Header.Get("User-Agent") != "NetTools/test" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"n":1}`))
		case "/limited":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		default:
			_, _ = w.Write([]byte(`not json`))
		}
	}))
	defer srv.Close()
	c := providers.New(target.NewResolver(target.Policy{AllowPrivate: true}), "NetTools/test")
	ctx := context.Background()

	var v struct{ N int }
	for range 3 {
		if err := c.GetJSON(ctx, providers.Request{Provider: "Test", URL: srv.URL + "/ok"}, &v); err != nil || v.N != 1 {
			t.Fatalf("v=%+v err=%v", v, err)
		}
	}
	if hits.Load() != 1 {
		t.Errorf("hits = %d, want 1 (cached)", hits.Load())
	}

	err := c.GetJSON(ctx, providers.Request{Provider: "Test", URL: srv.URL + "/limited?key=SECRET"}, &v)
	api := httpapi.ToAPIError(err)
	if api.Code != httpapi.CodeProviderError || !strings.Contains(api.Message, "rate limiting") || api.Details["status"] != 429 || strings.Contains(err.Error(), "SECRET") {
		t.Errorf("429: %+v (%v)", api, err)
	}
	if err := c.GetJSON(ctx, providers.Request{Provider: "Test", URL: srv.URL + "/missing"}, &v); !providers.IsNotFound(err) {
		t.Errorf("404: %v", err)
	}
	if err := c.GetJSON(ctx, providers.Request{Provider: "Test", URL: srv.URL + "/garbage"}, &v); httpapi.ToAPIError(err).Code != httpapi.CodeProviderError {
		t.Errorf("garbage: %v", err)
	}
}

// Transport errors must not echo the URL (it may carry an API key).
func TestErrorsRedactURL(t *testing.T) {
	c := providers.New(target.NewResolver(target.Policy{AllowPrivate: true}), "t")
	err := c.GetJSON(context.Background(), providers.Request{Provider: "Test", URL: "http://127.0.0.1:1/x?token=SECRET", TTL: -1}, &struct{}{})
	api := httpapi.ToAPIError(err)
	if err == nil || strings.Contains(err.Error(), "SECRET") || strings.Contains(api.Message, "SECRET") || api.Code != httpapi.CodeProviderError {
		t.Errorf("err = %v / %+v", err, api)
	}
	if api.Err != nil && strings.Contains(api.Err.Error(), "SECRET") {
		t.Errorf("logged cause leaks the key: %v", api.Err)
	}
}

// Provider hosts are dialed through the policy like any target.
func TestProviderPolicy(t *testing.T) {
	c := providers.New(target.NewResolver(target.Policy{}), "t")
	err := c.GetJSON(context.Background(), providers.Request{Provider: "Test", URL: "http://169.254.169.254/latest", TTL: -1}, &struct{}{})
	var te *target.Error
	if !errors.As(err, &te) {
		t.Errorf("err = %v", err)
	}
}
