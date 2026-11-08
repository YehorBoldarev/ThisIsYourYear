# syntax=docker/dockerfile:1

# Multi-arch: both build stages run on the builder's own platform
# ($BUILDPLATFORM) and the Go stage cross-compiles for the requested target
# ($TARGETOS/$TARGETARCH), so `docker buildx build --platform
# linux/amd64,linux/arm64 .` needs no emulation. A plain `docker build .` still
# works and builds for the local machine (e.g. natively on a Raspberry Pi).

# --- Frontend build stage ----------------------------------------------------
# Its output (HTML/JS/CSS) is architecture-independent, so it is built once.
FROM --platform=$BUILDPLATFORM node:24 AS web

WORKDIR /web

# Install deps against the lockfile first for better layer caching. .npmrc
# carries npm settings the install depends on (legacy-peer-deps, see the file).
COPY web/package.json web/package-lock.json web/.npmrc ./
RUN npm ci

COPY web/ ./
RUN npm run build

# --- Go build stage ----------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26 AS build

# Set automatically by BuildKit from --platform (or the local machine).
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Stage the frontend build output where `//go:embed all:dist` picks it up, then
# build the self-contained binary (SPA + API) with the embed build tag. The
# build is pure Go (CGO_ENABLED=0), so cross-compiling needs nothing extra.
COPY --from=web /web/dist/ ./internal/transport/http/webui/dist/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags embedui -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime stage -----------------------------------------------------------
# Resolved for the target platform; only the binary is copied in (no RUN), so
# no emulation is needed here either.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api

# Distroless nonroot already runs as an unprivileged user.
EXPOSE 8080
ENTRYPOINT ["/app/api"]
