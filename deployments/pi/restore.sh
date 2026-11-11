#!/usr/bin/env bash
# Restore the database from a backup made by backup.sh.
#
#   sudo ./restore.sh --drill <archive>   restore into a throwaway MongoDB and show
#                                         what's inside (live data untouched) — the
#                                         quarterly restore drill
#   sudo ./restore.sh <archive>           REPLACE the live database with the archive
#                                         (asks first; takes a safety backup first)
#   sudo ./restore.sh --yes <archive>     same, without the question
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$(readlink -f "$0")")/lib.sh"

usage() {
	sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
	exit 2
}

mode=live
assume_yes=false
case ${1-} in
--drill)
	mode=drill
	shift
	;;
--yes)
	assume_yes=true
	shift
	;;
-h | --help | "") usage ;;
esac
[[ $# -eq 1 ]] || usage
archive=$(readlink -f "$1")
[[ -f $archive ]] || die "no such file: $1"
gzip -t "$archive" || die "$archive is not a valid backup archive"

count_script="const d = db.getSiblingDB('$DB_NAME');
for (const c of ['users', 'topics', 'goals', 'sessions']) print(c.padEnd(9), d.getCollection(c).countDocuments());"

if [[ $mode == drill ]]; then
	drill=tiyy-restore-drill
	trap 'docker rm -f "$drill" >/dev/null 2>&1 || true' EXIT
	log "starting a throwaway MongoDB ($drill)"
	docker run -d --rm --name "$drill" "$(env_get MONGO_IMAGE "$DEFAULT_MONGO_IMAGE")" >/dev/null
	for _ in $(seq 1 30); do
		docker exec "$drill" mongosh --quiet --eval 'db.adminCommand("ping").ok' >/dev/null 2>&1 && break
		sleep 1
	done
	docker exec -i "$drill" mongorestore --quiet --archive --gzip <"$archive" || die "restore failed — this backup is NOT usable"
	log "restore drill ok — documents in $(basename "$archive"):"
	docker exec "$drill" mongosh --quiet --eval "$count_script"
	exit 0
fi

if [[ $assume_yes != true ]]; then
	echo "This REPLACES the live database with $(basename "$archive")."
	echo "Everything written since that backup will be lost (a safety backup is taken first)."
	read -r -p "Type 'restore' to continue: " answer
	[[ $answer == restore ]] || die "cancelled"
fi

log "taking a safety backup of the current data first"
"$KIT_DIR/backup.sh"

log "stopping the app during the restore"
compose stop api
log "restoring $(basename "$archive")"
if ! compose exec -T mongo mongorestore --quiet --archive --gzip --drop --nsInclude "$DB_NAME.*" <"$archive"; then
	compose start api
	die "restore failed; the app was restarted (data may be partially restored — restore the safety backup)"
fi
compose start api
wait_healthy 120 || die "the app didn't come back healthy after the restore — check: docker compose logs api"
log "restore complete:"
compose exec -T mongo mongosh --quiet --eval "$count_script"
