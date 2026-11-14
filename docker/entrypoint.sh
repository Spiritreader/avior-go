#!/bin/sh
set -e
# Copy binary into persistent data directory so ReflectionPath() (= directory of the
# executable) resolves to /data and config.json + log/ survive container restarts.
#
# /data/avior-go may be the currently running binary (executed from this volume on a
# previous start). "Text file busy" blocks plain cp onto a running executable, so
# remove first, then copy. rm on a running binary is fine on Linux (inode stays alive
# until the process exits); the new file is created fresh afterwards.
rm -f /data/avior-go
cp /opt/avior-go-bin /data/avior-go
chmod +x /data/avior-go
cd /data
# Files the app creates must stay readable/writable for every authorized user on
# the host (SMB/NFS), matching the other Unraid tools (add_rating_ui sets umask 0
# for the same reason). umask only removes permission bits, so loggers that open
# their file with 0600 (lumberjack) are additionally chmodded by the app.
umask 000
# PUID/PGID (LinuxServer convention): run the app as the configured user/group so
# every file it creates (logs in /data/log, .INFO.log next to media, config.json)
# is owned by that UID/GID on the host. Default: root (no env set).
if [ -n "$PUID" ] && [ -n "$PGID" ]; then
  # Ensure the target directories are owned by the configured user.
  chown -R "$PUID:$PGID" /data 2>/dev/null || true
  # setpriv (util-linux) is preinstalled on the Ubuntu-based ffmpeg image; su-exec
  # is an Alpine package and NOT available here, but keep it as a fallback.
  if command -v setpriv >/dev/null 2>&1; then
    exec setpriv --reuid="$PUID" --regid="$PGID" --clear-groups /data/avior-go
  elif command -v su-exec >/dev/null 2>&1; then
    exec su-exec "$PUID:$PGID" /data/avior-go
  else
    echo "warning: neither setpriv nor su-exec found, running as root" >&2
  fi
fi
exec /data/avior-go
