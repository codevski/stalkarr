#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "
─────────────────────────────────────
        stalkarr
─────────────────────────────────────
User UID: ${PUID}
User GID: ${PGID}
─────────────────────────────────────
"

if ! getent group stalkarr > /dev/null 2>&1; then
    addgroup -g "$PGID" stalkarr
fi

if ! getent passwd stalkarr > /dev/null 2>&1; then
    adduser -D -u "$PUID" -G stalkarr stalkarr
fi

mkdir -p /config
chown -R stalkarr:stalkarr /config
chown -R stalkarr:stalkarr /app

exec su-exec stalkarr:stalkarr /app/stalkarr
