# syntax=docker/dockerfile:1
FROM node:24-alpine AS build
WORKDIR /src
# pnpm version comes from "packageManager" in package.json
RUN corepack enable
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/web/package.json apps/web/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter @nettools/web
COPY apps/web/ apps/web/
ARG VERSION=dev
ENV VITE_APP_VERSION=$VERSION
RUN pnpm --filter @nettools/web build

FROM nginxinc/nginx-unprivileged:stable-alpine
# official nginx images render /etc/nginx/templates/*.template with envsubst at start
COPY docker/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY docker/nginx/security-headers.conf /etc/nginx/snippets/security-headers.conf
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
ENV NGINX_ENVSUBST_FILTER="^(API_UPSTREAM|ENABLE_HSTS)$" \
    API_UPSTREAM=http://api:8081 \
    ENABLE_HSTS=false
EXPOSE 8080
