# Docker Deployment for avior-go

## Setup

1. Copy `docker/config.example.json` to `/mnt/user/appdata/avior-go/config.json` on your
   host and adjust settings (MongoDB/Redis hosts, paths).

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

The container uses the `linuxserver/ffmpeg:latest` base image which ships ffmpeg with
Intel QSV (oneVPL) support. On an Unraid host with an Intel ARC GPU:

- The device is passed through via `devices: ["/dev/dri:/dev/dri"]` in `compose.yaml`
- `av1_qsv`, `-hwaccel qsv`, `-hwaccel_output_format qsv` work out of the box
- The example config (`docker/config.example.json`) is pre-configured for QSV

### Software fallback

If no GPU is available, replace the encoder in `config.json`:

| QSV (hardware)          | Software equivalent     |
|-------------------------|-------------------------|
| `av1_qsv`               | `libsvtav1` or `libaom-av1` |
| `-hwaccel qsv`          | (remove)                |
| `-hwaccel_output_format qsv` | (remove)           |
| `-init_hw_device qsv=qsv` | (remove)              |
| `-filter_hw_device qsv`  | (remove)               |

And switch the runtime base in `Dockerfile` back to `debian:bookworm-slim` with
additional packages:
```dockerfile
RUN apt-get install -y libsvtav1enc1
```



## Volume layout

| Host path                        | Container path | Purpose              |
|----------------------------------|---------------|----------------------|
| `/mnt/user/media`                | `/media`      | Media library        |
| `/mnt/user/appdata/avior-go`     | `/data`       | config.json + logs   |
