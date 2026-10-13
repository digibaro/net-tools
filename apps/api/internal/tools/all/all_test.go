package all

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
)

// Every route a tool mounts must be documented in openapi.yaml (CLAUDE.md rule 9), and
// capability ids must be unique.
func TestOpenAPICoversAllRoutes(t *testing.T) {
	spec, err := os.ReadFile("../../../openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFrom(func(string) (string, bool) { return "", false }, func(string) ([]byte, error) { return nil, io.EOF })
	if err != nil {
		t.Fatal(err)
	}
	o := httpapi.Options{Config: cfg, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	deps := httpapi.BuildDeps(&o)

	seen := map[string]bool{}
	r := chi.NewRouter()
	for _, tool := range Tools() {
		if seen[tool.ID()] {
			t.Errorf("duplicate capability id %s", tool.ID())
		}
		seen[tool.ID()] = true
		tool.Mount(r, deps)
		if !strings.Contains(string(spec), "x-capability: "+tool.ID()+"\n") {
			t.Errorf("capability %s not referenced by any operation in openapi.yaml", tool.ID())
		}
	}
	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !strings.Contains(string(spec), "\n  /v1"+route+":\n") {
			t.Errorf("%s /v1%s is not documented in openapi.yaml", method, route)
		}
		return nil
	})

}
