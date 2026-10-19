# Operations

## Configuration (environment variables)
| Variable | Meaning | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | local dev URL |
| `JWT_SECRET` | signing key, **must be changed in production** | dev value |
| `PORT` | HTTP port | 8080 |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | bootstrap administrator, created at start if missing | - |
| `COOKIE_SECURE` | `true` behind HTTPS | false |
| `LOGIN_RATE_LIMIT` | login attempts per minute per IP | 10 |
| `REMINDER_HOUR` | hour of the attendance reminder, -1 disables | 15 |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | e-mail delivery | off |
| `EMAIL_LOG` | `true` writes e-mails to the log (demo) | false |

## Run
`docker compose up --build`, then apply migrations (`make migrate-up`) and optionally the demo data (`go run ./cmd/seed`).

## Backup
```
DATABASE_URL=postgres://... BACKUP_DIR=/var/backups/school BACKUP_KEEP_DAYS=14 scripts/backup.sh
```
The script writes a compressed custom-format dump (`pg_dump --format=custom`) to a temporary file, renames it when finished, checks it with `pg_restore --list` and deletes dumps older than `BACKUP_KEEP_DAYS`. Run it daily:
```
15 2 * * *  cd /opt/school-management && DATABASE_URL=... scripts/backup.sh >> /var/log/school-backup.log 2>&1
```
Store a copy outside the server (another host or object storage).

## Restore
```
createdb school_restored
DATABASE_URL=postgres://.../school_restored scripts/restore.sh /var/backups/school/<file>.dump
```
The target database must be empty; the script stops on the first error. Check the result with `SELECT count(*) FROM users;` and start the application against it. Do a restore drill at least once per term: a backup that was never restored is not a backup.

Targets: RPO 24 h (daily dump), RTO under 30 min for a school-sized database.
