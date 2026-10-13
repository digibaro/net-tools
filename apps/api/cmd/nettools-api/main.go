// Command nettools-api is the NetTools backend.
//
//	nettools-api [serve]     run the HTTP server (default)
//	nettools-api healthcheck probe /api/healthz on the local listener (Docker HEALTHCHECK)
//	nettools-api version     print build information
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/config"
	"github.com/digibaro/net-tools/apps/api/internal/geoip"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools/all"
)

// Set via -ldflags "-X main.version=… -X main.commit=… -X main.buildDate=…".
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	var err error
	switch cmd {
	case "serve":
		err = serve()
	case "healthcheck":
		err = healthcheck()
	case "version":
		_, err = fmt.Fprintf(os.Stdout, "nettools-api %s (commit %s, built %s, %s)\n",
			version, commit, buildDate, runtime.Version())
	default:
		err = fmt.Errorf("unknown command %q (want serve, healthcheck or version)", cmd)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "nettools-api:", err)
		os.Exit(1)
	}
}

func serve() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	resolver := target.NewResolver(target.Policy{
		AllowPrivate: cfg.Features.AllowPrivate,
		Deny:         cfg.Features.DenyCIDRs,
		Allow:        cfg.Features.AllowCIDRs,
		Self:         selfAddrs(logger),
	})

	handler := httpapi.NewRouter(httpapi.Options{
		Config: cfg,
		Logger: logger,
		Target: resolver,
		Build: httpapi.BuildInfo{
			Version: version, Commit: commit, BuildDate: buildDate, GoVersion: runtime.Version(),
		},
		Tools: all.Tools(),
		Ready: []httpapi.ReadyCheck{{Name: "geoip", Check: func(context.Context) error {
			return geoip.ReadyCheck(cfg.DataDir, cfg.Features.GeoIPCityDB, cfg.Features.GeoIPASNDB)
		}}},
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// No WriteTimeout: SSE streams run up to Limits.StreamTimeoutMax and are bounded per handler.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 32 << 10,
		BaseContext:    func(net.Listener) context.Context { return ctx },
		ErrorLog:       slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", slog.String("addr", cfg.Listen), slog.String("profile", string(cfg.Profile)),
			slog.String("version", version))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func healthcheck() error {
	listen := os.Getenv("API_LISTEN")
	if listen == "" {
		listen = "0.0.0.0:8081"
	}
	url, err := healthURL(listen)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// url is a loopback address derived from API_LISTEN, never from a request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) //nolint:gosec // see above
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // loopback healthcheck
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}

// healthURL maps a listen address to a loopback URL (wildcard hosts become 127.0.0.1).
func healthURL(listen string) (string, error) {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", fmt.Errorf("API_LISTEN %q: %w", listen, err)
	}
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/api/healthz", nil
}

// selfAddrs returns this host's interface addresses; targets matching them count as private.
func selfAddrs(logger *slog.Logger) []netip.Addr {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Warn("cannot list interface addresses", slog.Any("err", err))
		return nil
	}
	var out []netip.Addr
	for _, a := range addrs {
		if p, err := netip.ParsePrefix(a.String()); err == nil {
			out = append(out, p.Addr())
		}
	}
	return out
}
