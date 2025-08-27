#!/bin/sh
set -eu

[ -n "${UMASK:-}" ] && umask "$UMASK"

if [ "$(id -u)" = '0' ]; then
  PUID=${PUID:-1000}
  PGID=${PGID:-1000}

  DATA_DIRS="/app /usr/local/bin"
  for DIR in ${DATA_DIRS}; do
    if [ -d "$DIR" ]; then
      chown -R "${PUID}:${PGID}" "$DIR" || true
    fi
  done

  exec gosu "${PUID}:${PGID}" "$@"
else
  exec "$@"
fi
