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

## Hardware acceleration

The example config uses Intel QSV (`av1_qsv` encoder, `-hwaccel qsv`). On Linux this
requires:

- Intel GPU with `/dev/dri` available
- ffmpeg built with `--enable-libvpl` or `--enable-libmfx` (not in stock Debian ffmpeg)

To enable GPU passthrough, uncomment the `devices` section in `compose.yaml`.

### Software fallback

If no GPU is available, replace the encoder in `config.json`:

| QSV (hardware)          | Software equivalent     |
|-------------------------|-------------------------|
| `av1_qsv`               | `libsvtav1` or `libaom-av1` |
| `-hwaccel qsv`          | (remove)                |
| `-hwaccel_output_format qsv` | (remove)           |
| `-init_hw_device qsv=qsv` | (remove)              |
| `-filter_hw_device qsv`  | (remove)               |

And install the codec in the Dockerfile:
```dockerfile
RUN apt-get install -y libsvtav1enc1
```

## Volume layout

| Host path                        | Container path | Purpose              |
|----------------------------------|---------------|----------------------|
| `/mnt/user/media`                | `/media`      | Media library        |
| `/mnt/user/appdata/avior-go`     | `/data`       | config.json + logs   |
