#!/bin/sh
# Backs up the school database with pg_dump (custom format, compressed) and deletes old backups.
#
#   DATABASE_URL=postgres://user:pass@host:5432/school_management ./scripts/backup.sh [backup_dir]
#
# Environment:
#   DATABASE_URL     connection string (default: the local development database)
#   BACKUP_DIR       target directory (default: ./backups, or the first argument)
#   BACKUP_KEEP_DAYS backups older than this many days are removed (default: 14)
#
# Run it from cron, for example every night at 02:30:
#   30 2 * * * cd /opt/school-management && DATABASE_URL=... ./scripts/backup.sh /var/backups/school
set -eu

DATABASE_URL="${DATABASE_URL:-postgres://postgres:password@localhost:5432/school_management?sslmode=disable}"
BACKUP_DIR="${1:-${BACKUP_DIR:-./backups}}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"

mkdir -p "$BACKUP_DIR"
file="$BACKUP_DIR/school_management_$(date +%Y%m%d_%H%M%S).dump"

# write to a temporary name first so a half-finished dump is never mistaken for a good backup
pg_dump --format=custom --no-owner --dbname="$DATABASE_URL" --file="$file.partial"
mv "$file.partial" "$file"

# a backup that cannot be listed back is useless: verify it
pg_restore --list "$file" > /dev/null

find "$BACKUP_DIR" -name 'school_management_*.dump' -type f -mtime +"$KEEP_DAYS" -delete

echo "Backup written: $file ($(du -h "$file" | cut -f1))"
