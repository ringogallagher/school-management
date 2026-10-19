#!/bin/sh
# Restores a backup made by backup.sh into an EMPTY database (it never drops anything).
#
#   DATABASE_URL=postgres://user:pass@host:5432/restored_db ./scripts/restore.sh backups/school_management_20261013_023000.dump
#
# Create the target first:  createdb restored_db
set -eu

if [ $# -ne 1 ]; then
  echo "usage: $0 <backup.dump>" >&2
  exit 2
fi
DATABASE_URL="${DATABASE_URL:?set DATABASE_URL to the (empty) database to restore into}"

pg_restore --no-owner --exit-on-error --dbname="$DATABASE_URL" "$1"
echo "Restored $1"
