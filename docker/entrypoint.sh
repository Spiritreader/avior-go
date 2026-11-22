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
LIB_REPO="${MOVIE_NFO_LIB_REPO:-https://github.com/Spiritreader/movie_nfo_lib.git}"
LIB_REF="${MOVIE_NFO_LIB_REF:?MOVIE_NFO_LIB_REF is required}"
LIB_DIR=/data/.vendor/movie_nfo_lib
if [ -z "${GIT_TOKEN:-}" ]; then
  echo "GIT_TOKEN is required to provision movie_nfo_lib" >&2
  exit 1
fi
rm -rf "$LIB_DIR"
mkdir -p "$(dirname "$LIB_DIR")"
ASKPASS=$(mktemp)
trap 'rm -f "$ASKPASS"' EXIT
cat >"$ASKPASS" <<'EOF'
#!/bin/sh
LIB_REPO="${MOVIE_NFO_LIB_REPO:-https://github.com/Spiritreader/movie_nfo_lib.git}"
LIB_REF="${MOVIE_NFO_LIB_REF:?MOVIE_NFO_LIB_REF is required}"
LIB_DIR=/data/.vendor/movie_nfo_lib
if [ -z "${GIT_TOKEN:-}" ]; then
  echo "GIT_TOKEN is required to provision movie_nfo_lib" >&2
  exit 1
fi
rm -rf "$LIB_DIR"
mkdir -p "$(dirname "$LIB_DIR")"
ASKPASS=$(mktemp)
trap 'rm -f "$ASKPASS"' EXIT
cat >"$ASKPASS" <<'EOF'
#!/bin/sh
printf '%s\n' "$GIT_TOKEN"
EOF
chmod 700 "$ASKPASS"
GIT_ASKPASS="$ASKPASS" GIT_TERMINAL_PROMPT=0 git -c credential.username=x-access-token clone --quiet --depth 1 --branch "$LIB_REF" "$LIB_REPO" "$LIB_DIR"
python3 -m venv /data/.vendor/venv
/data/.vendor/venv/bin/pip install --disable-pip-version-check --quiet -r /opt/avior-go-llm-compare/requirements.txt
/data/.vendor/venv/bin/pip install --disable-pip-version-check --quiet "$LIB_DIR"
export PYTHONPATH="$LIB_DIR${PYTHONPATH:+:$PYTHONPATH}"
export AVIOR_LLM_ADAPTER=/opt/avior-go-llm-compare/compare.py

 # Copy binary into persistent data directory so ReflectionPath() (= directory of the
