#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "
─────────────────────────────────────
sleeparr
─────────────────────────────────────
User UID: ${PUID}
User GID: ${PGID}
─────────────────────────────────────
"

if ! getent group sleeparr > /dev/null 2>&1; then
    addgroup -g "$PGID" sleeparr
fi

if ! getent passwd sleeparr > /dev/null 2>&1; then
    adduser -D -u "$PUID" -G sleeparr sleeparr
fi

mkdir -p /config
chown -R sleeparr:sleeparr /config
chown -R sleeparr:sleeparr /app

exec su-exec sleeparr:sleeparr /app/sleeparr
