#!/usr/bin/env bash
# Update the app to the newest published image, safely:
#   pull → (nothing new? stop) → backup → restart on the new image → wait for
#   healthy → if it isn't healthy within 2 minutes, roll back to the previous
#   image and exit with an error.
#
#   sudo ./update.sh            (also run nightly at 04:00 by tiyy-update.timer)
#
# Only the app image is updated here. MongoDB is pinned in .env and bumped by
# hand (docs/operations.md, monthly checklist).
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$(readlink -f "$0")")/lib.sh"

readonly ROLLBACK_TAG=tiyy-api:rollback

cid=$(compose ps -q api)
[[ -n $cid ]] || die "the app isn't running — start it first: docker compose up -d"
old_image=$(docker inspect --format '{{.Image}}' "$cid")

# Keep a handle on the version that works now, in case the new one doesn't.
# This must happen BEFORE pulling: with Docker's containerd image store, an
# image that loses its last tag (as the old :latest does on pull) is deleted.
docker tag "$old_image" "$ROLLBACK_TAG" ||
	die "can't keep a rollback copy of the running version (it was probably replaced by a manual 'docker compose pull'). Nothing was changed. Run 'docker compose up -d' to start the already-pulled version, then rerun update.sh"

app_image=$(env_get APP_IMAGE "$DEFAULT_APP_IMAGE")
log "checking for a new version of $app_image"
compose pull --quiet api || die "pull failed (no internet, or the registry is down) — nothing was changed"
new_image=$(docker image inspect --format '{{.Id}}' "$app_image")
if [[ $new_image == "$old_image" ]]; then
	log "already up to date"
	exit 0
fi

"$KIT_DIR/backup.sh"

log "starting the new version"
compose up -d api
if wait_healthy 120; then
	log "update ok — the new version is healthy"
	docker image prune -f >/dev/null # drop superseded, untagged images
	exit 0
fi

log "the new version is NOT healthy — rolling back"
compose logs --tail 40 api >&2 || true
APP_IMAGE=$ROLLBACK_TAG compose up -d api
wait_healthy 120 || die "the previous version is unhealthy too — see docs/operations.md, Troubleshooting"
die "update rolled back; the previous version is running. It will retry tomorrow — to stop that, pin a known-good tag in .env (APP_IMAGE=…:sha-…)"
