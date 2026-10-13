# syntax=docker/dockerfile:1
# Use the latest stable Go; bump via Renovate.
ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
      -o /out/nettools-api ./cmd/nettools-api

FROM debian:trixie-slim
ARG INSTALL_NMAP=false
RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      ca-certificates tini mtr-tiny traceroute iperf3 bgpq4; \
    if [ "$INSTALL_NMAP" = "true" ]; then apt-get install -y --no-install-recommends nmap; fi; \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /out/nettools-api /usr/local/bin/nettools-api
ENV API_LISTEN=0.0.0.0:8081 DATA_DIR=/data
EXPOSE 8081
# Runs as root inside the container with ONLY CAP_NET_RAW (compose drops all others),
# read-only rootfs and no-new-privileges. See docs/SECURITY.md §5 for the non-root alternative.
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/nettools-api"]
CMD ["serve"]
