# syntax=docker/dockerfile:1

# --- Frontend build stage ----------------------------------------------------
FROM node:22 AS web

WORKDIR /web

# Install deps against the lockfile first for better layer caching.
COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# --- Go build stage ----------------------------------------------------------
FROM golang:1.26 AS build

WORKDIR /src

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Stage the frontend build output where `//go:embed all:dist` picks it up, then
# build the self-contained binary (SPA + API) with the embed build tag.
COPY --from=web /web/dist/ ./internal/transport/http/webui/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -tags embedui -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api

# Distroless nonroot already runs as an unprivileged user.
EXPOSE 8080
ENTRYPOINT ["/app/api"]
