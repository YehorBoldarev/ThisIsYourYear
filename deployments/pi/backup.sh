#!/usr/bin/env bash
# Back up the database to $BACKUP_DIR and delete archives beyond $BACKUP_KEEP.
#
#   sudo ./backup.sh            (also run daily at 03:30 by tiyy-backup.timer)
#
# Output: $BACKUP_DIR/tiyy-YYYYMMDD-HHMMSS.archive.gz (a gzip-compressed
# mongodump archive; restore it with ./restore.sh).
set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "$(readlink -f "$0")")/lib.sh"

backup_dir=$(env_get BACKUP_DIR /var/backups/thisisyouryear)
keep=$(env_get BACKUP_KEEP 14)
[[ $keep =~ ^[0-9]+$ && $keep -ge 1 ]] || die "BACKUP_KEEP must be a whole number >= 1 (got '$keep')"

mkdir -p "$backup_dir"
chmod 700 "$backup_dir"

target="$backup_dir/tiyy-$(date '+%Y%m%d-%H%M%S').archive.gz"
partial="$target.partial"
trap 'rm -f "$partial"' EXIT

log "backing up database '$DB_NAME' to $target"
compose exec -T mongo mongodump --quiet --archive --gzip --db "$DB_NAME" >"$partial" ||
	die "mongodump failed — is the stack running? (docker compose ps)"
[[ -s $partial ]] || die "the backup came out empty"
gzip -t "$partial" || die "the backup archive is corrupt"
mv "$partial" "$target"
chmod 600 "$target"
log "backup ok: $(du -h "$target" | cut -f1)"

# Keep the newest $keep archives. Names sort chronologically.
shopt -s nullglob
archives=("$backup_dir"/tiyy-*.archive.gz)
excess=$((${#archives[@]} - keep))
for ((i = 0; i < excess; i++)); do
	rm -f -- "${archives[i]}"
	log "deleted old backup $(basename "${archives[i]}")"
done
log "$((${#archives[@]} - (excess > 0 ? excess : 0))) backup(s) kept in $backup_dir"
