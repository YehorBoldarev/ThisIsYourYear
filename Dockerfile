# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
FROM golang:1.26 AS build

WORKDIR /src

# Cache dependencies separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static, stripped binary so it runs on the distroless base.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# --- Runtime stage -----------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/api /app/api

# Distroless nonroot already runs as an unprivileged user.
EXPOSE 8080
ENTRYPOINT ["/app/api"]
