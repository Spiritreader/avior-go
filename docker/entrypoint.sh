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
exec /data/avior-go
