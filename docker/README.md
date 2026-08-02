# Docker Deployment for avior-go

## Setup

Two config variants exist:

- `docker/config.example.json` — **QSV hardware encoding** (Intel ARC GPU, needs
  `/dev/dri` passthrough)
- `docker/config.software.example.json` — **software encoding** (`libsvtav1`, runs
  without any GPU)

1. Copy the config matching your hardware to `/mnt/user/appdata/avior-go/config.json`
   on your host and adjust settings (MongoDB/Redis hosts, paths).

2. Path convention: the compose file mounts `/mnt/user/media` to `/media` inside the
   container. All `MediaPaths`, `ObsoletePath`, and `EncoderConfig.*.OutDirectory`
   values must use `/media/...` paths (forward slashes, no UNC).

3. Deploy via Komodo from this repo's compose stack, or manually:
   ```
   docker compose up -d
   ```

### Komodo stack config

Create a Stack in Komodo pointing at the git repo and **force a rebuild on every
redeploy** — otherwise `docker compose up` reuses a cached image with the same name
and code changes stay invisible:

```toml
[[stack]]
name = "avior-go"
[stack.config]
server = "<your-unraid-server>"
run_directory = "/opt/stacks/avior-go"
file_paths = ["compose.yaml"]
repo = "Spiritreader/avior-go"
branch = "feat/dual-os"   # compose/Dockerfile currently live here
# rebuild the image on every deploy (no image: tag in compose.yaml):
extra_args = "--build"
```

## Hardware acceleration

### Without GPU (software encoding) — default for this branch

The container uses the `linuxserver/ffmpeg:latest` base image (a full ffmpeg build
including `libsvtav1` and `libopus`). Without a GPU, use
`docker/config.software.example.json` — all QSV args removed, `av1_qsv` replaced by
`libsvtav1` (`-preset 8 -crf 26`). No `/dev/dri` passthrough needed.

### With Intel ARC GPU (QSV)

`linuxserver/ffmpeg` ships Intel QSV (oneVPL) support. To enable:

1. Uncomment/add `devices: ["/dev/dri:/dev/dri"]` in `compose.yaml`
   (requires the i915 driver loaded on the host — see `ls /dev/dri`)
2. Use `docker/config.example.json` (pre-configured for `av1_qsv`)

| QSV (hardware)          | Software equivalent (`config.software.example.json`) |
|-------------------------|------------------------------------------------------|
| `av1_qsv`               | `libsvtav1`                                          |
| `-preset veryslow`      | `-preset 8`                                          |
| `-global_quality:v 26`  | `-crf 26`                                            |
| `-hwaccel qsv` / `-hwaccel_output_format qsv` / `-init_hw_device qsv=qsv` / `-filter_hw_device qsv` | (removed) |



## Volume layout

| Host path                        | Container path | Purpose              |
|----------------------------------|---------------|----------------------|
| `/mnt/user/media`                | `/media`      | Media library        |
| `/mnt/user/appdata/avior-go`     | `/data`       | config.json + logs   |
