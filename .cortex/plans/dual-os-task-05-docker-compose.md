# Task 05: Dockerfile + compose.yaml für Komodo-Deployment

## Ziel

avior-go läuft als Docker-Container unter Linux, verwaltet über Komodo (Stack aus dem
Repo). Host-Verzeichnis `/mnt/user/media` wird als `/media` in den Container gemountet.

## Verifizierte Rahmenbedingungen aus dem Code

- `globalstate.ReflectionPath()` (globalstate/globalstate.go:45) liefert
  `filepath.Dir(os.Executable())` — dorthin werden `config.json` gelesen/geschrieben
  (`config/config.go:259,290-303`) und `log/*.log` angelegt (app.go:34-36).
  Konsequenz: Das Binary muss in einem beschreibbaren, persistenten Verzeichnis liegen,
  damit Config und Logs Container-Neustarts überleben.
- HTTP-API-Port: `10000 + cfg.Local.Instance` (api/api.go:119) → Default **10000**.
- Externe Abhängigkeiten: MongoDB (`DatabaseURL`, Default `mongodb://localhost:27017`)
  und optional Redis (`config.go:177-182`, `Enabled: false` per Default). Beide laufen
  bereits extern (configs zeigen auf `10.10.10.96`) → **keine** mongo/redis-Services
  ins compose aufnehmen. Die `config.json` der Linux-Instanz muss auf die externe
  MongoDB zeigen und Linux-Pfade (`/media/...`) für `MediaPaths`/`OutDirectory`/`ObsoletePath`
  verwenden — das ist Benutzerkonfiguration, kein Code.
- ffmpeg/ffprobe werden per `exec.Command("ffmpeg", ...)` / `exec.Command("ffprobe", ...)`
  aus dem PATH aufgerufen (encoder/encoder.go:167, tools/tools.go:118,137) → Image
  braucht ffmpeg.

## Edits

### 1. Neue Datei `Dockerfile` im Repo-Root

Zweistufig: Build-Stage kompiliert das Linux-Binary (ersetzt den manuellen
`make build-linux`-Schritt für Docker), Runtime-Stage ist Debian-slim mit ffmpeg.

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /out/avior-go app.go

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/avior-go /opt/avior-go-bin
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
```

### 2. Neue Datei `docker/entrypoint.sh`

Hintergrund: `ReflectionPath()` = Verzeichnis des Executable. Config/Logs müssen im
persistenten Volume `/data` liegen, also kopiert der Entrypoint das Binary dorthin und
startet es aus `/data`. Beim ersten Start wird keine config.json angelegt — die App
erzeugt sie selbst mit Defaults (`config.Instance()`/`TryMakeCopy`), die danach auf dem
Host editierbar sind.

```sh
#!/bin/sh
set -e
# Binary ins persistente Datenverzeichnis kopieren, damit ReflectionPath() (= Dir des
# Executable) auf /data zeigt und config.json + log/ Container-Neustarts überleben.
cp /opt/avior-go-bin /data/avior-go
chmod +x /data/avior-go
cd /data
exec /data/avior-go
```

### 3. Neue Datei `compose.yaml` im Repo-Root (Komodo-kompatibel)

Komodo baut den Stack direkt aus dem Repo (`build: .`), kein Registry-Push nötig.

```yaml
services:
  avior-go:
    build: .
    image: avior-go:latest
    container_name: avior-go
    restart: unless-stopped
    ports:
      - "10000:10000"
    volumes:
      - /mnt/user/media:/media
      - /mnt/user/appdata/avior-go:/data
    cap_add:
      - SYS_NICE   # erlaubt negative nice-Werte (EncoderPriority HIGH/ABOVE_NORMAL);
                   # ohne Capability läuft der Encode trotzdem, nur mit Warnung im Log
```

Entscheidungen:
- `/mnt/user/appdata/avior-go` als Host-Pfad für `/data` (Unraid-Konvention für
  App-Daten; Komodo-Ziel ist laut Mount-Pfad `/mnt/user/...` ein Unraid-Host).
  Falls das Verzeichnis nicht existiert, legt Docker es beim ersten Start an.
- `.dockerignore` neu anlegen mit `dist/`, `*.exe`, `.git/`, `log/`, damit der
  Build-Kontext klein bleibt.

### 4. config.json für die Linux-Instanz (Benutzeraktion nach erstem Start)

Nach dem ersten Start liegt `/mnt/user/appdata/avior-go/config.json` auf dem Host.
Darin Pfade auf Linux-Form anpassen: `MediaPaths`, `ObsoletePath`,
`EncoderConfig.*.OutDirectory` → z. B. `/media/transcoded`, `/media/tv`
(entsprechen den bisherigen UNC-Pfaden `\\UMS\media\...`, da `/mnt/user/media` der
gemountete Share ist). `DatabaseURL` auf die erreichbare MongoDB setzen.

## Check

- `docker build -t avior-go:test .` aus dem Repo-Root läuft durch.
- `docker run --rm -v /tmp/avior-data:/data avior-go:test` startet; in
  `/tmp/avior-data` erscheinen `avior-go`, `config.json`, `log/`. API antwortet:
  `curl http://localhost:10000/` (Port gemappt per `-p 10000:10000`).
- Im Container: `docker exec <c> ffmpeg -version` findet ffmpeg.

## Abhängigkeit

Setzt Task 01 voraus (Linux kompilierbar) und nutzt denselben Build-Befehl wie Task 02.
