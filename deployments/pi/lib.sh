#!/usr/bin/env bash
# Shared helpers for the Pi scripts (backup.sh, restore.sh, update.sh).
# This file is sourced, not executed.

# The kit directory (where this file, docker-compose.yml and .env live).
KIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly KIT_DIR
# These are used by the scripts that source this file.
# shellcheck disable=SC2034
readonly DB_NAME=thisisyouryear
# shellcheck disable=SC2034
readonly DEFAULT_APP_IMAGE=ghcr.io/yehorboldarev/thisisyouryear:latest
# shellcheck disable=SC2034
readonly DEFAULT_MONGO_IMAGE=mongo:7.0.45

log() { printf '%s %s\n' "$(date '+%F %T')" "$*"; }
die() {
	log "ERROR: $*" >&2
	exit 1
}

# env_get KEY DEFAULT — read KEY from .env without executing the file. An unset
# or empty value yields DEFAULT.
env_get() {
	local key=$1 default=${2-} line value
	line=$(grep -E "^${key}=" "$KIT_DIR/.env" 2>/dev/null | tail -n 1 || true)
	value=${line#*=}
	value=${value%\"}
	value=${value#\"}
	if [[ -n $value ]]; then
		printf '%s' "$value"
	else
		printf '%s' "$default"
	fi
}

# compose … — docker compose for this kit, from any working directory.
compose() {
	docker compose --project-directory "$KIT_DIR" -f "$KIT_DIR/docker-compose.yml" "$@"
}

# api_health — the api container's health status (healthy, starting,
# unhealthy), or "missing" when it isn't running.
api_health() {
	local cid
	cid=$(compose ps -q api 2>/dev/null || true)
	if [[ -z $cid ]]; then
		echo missing
		return
	fi
	docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$cid" 2>/dev/null || echo missing
}

# wait_healthy SECONDS — wait until the api container reports healthy.
wait_healthy() {
	local timeout=$1 waited=0
	while ((waited < timeout)); do
		[[ $(api_health) == healthy ]] && return 0
		sleep 3
		waited=$((waited + 3))
	done
	return 1
}

[[ -f "$KIT_DIR/.env" ]] || die "$KIT_DIR/.env is missing — copy .env.example to .env and fill it in (docs/operations.md)"
