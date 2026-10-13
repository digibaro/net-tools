# syntax=docker/dockerfile:1
FROM node:22-alpine AS build
WORKDIR /src/apps/web
RUN corepack enable
COPY apps/web/package.json apps/web/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY apps/web/ ./
COPY apps/api/openapi.yaml /src/apps/api/openapi.yaml
ARG VERSION=dev
ARG COMMIT=unknown
ENV VITE_APP_VERSION=$VERSION VITE_APP_COMMIT=$COMMIT
RUN pnpm build

FROM nginxinc/nginx-unprivileged:stable-alpine
# official nginx images render /etc/nginx/templates/*.template with envsubst at start
COPY docker/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/apps/web/dist /usr/share/nginx/html
ENV NGINX_ENVSUBST_FILTER="^(API_UPSTREAM|ENABLE_HSTS)$"
EXPOSE 8080
