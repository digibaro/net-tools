# syntax=docker/dockerfile:1
# Static output is platform independent: build natively, never under QEMU.
FROM --platform=$BUILDPLATFORM node:24-alpine AS build
WORKDIR /src
# pnpm version comes from "packageManager" in package.json
RUN corepack enable
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/web/package.json apps/web/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter @nettools/web
COPY apps/web/ apps/web/
ARG VERSION=dev
ARG LATEST_RELEASE=
ENV VITE_APP_VERSION=$VERSION VITE_LATEST_RELEASE=$LATEST_RELEASE
RUN pnpm --filter @nettools/web build
# Allow exactly our inline scripts (anti-FOUC theme script) in the CSP.
COPY docker/nginx/security-headers.conf /src/security-headers.conf
RUN hashes="$(cat apps/web/dist/csp-script-hashes.txt)" \
    && sed -i "s|script-src 'self';|script-src 'self' ${hashes};|" /src/security-headers.conf \
    && grep -q "script-src 'self' 'sha256-" /src/security-headers.conf \
    && rm apps/web/dist/csp-script-hashes.txt
# Precompress static assets at maximum level; nginx serves them with gzip_static.
RUN find apps/web/dist -type f \( -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.svg' \
      -o -name '*.json' -o -name '*.webmanifest' -o -name '*.txt' \) -size +1k -exec gzip -9 -k -n {} +

FROM nginxinc/nginx-unprivileged:stable-alpine
# official nginx images render /etc/nginx/templates/*.template with envsubst at start
COPY docker/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
ENV NGINX_ENVSUBST_FILTER="^(API_UPSTREAM|ENABLE_HSTS)$" \
    API_UPSTREAM=http://api:8081 \
    ENABLE_HSTS=false
EXPOSE 8080
