#!/bin/sh
set -e
# Copy binary into persistent data directory so ReflectionPath() (= directory of the
# executable) resolves to /data and config.json + log/ survive container restarts.
cp /opt/avior-go-bin /data/avior-go
chmod +x /data/avior-go
cd /data
exec /data/avior-go
