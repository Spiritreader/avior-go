# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o /out/avior-go app.go
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-venv git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY llm_compare/requirements.txt /tmp/avior-requirements.txt

RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 python3-venv git ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY llm_compare/requirements.txt /tmp/avior-requirements.txt

ARG MOVIE_NFO_LIB_REF=v1.0.79
ENV MOVIE_NFO_LIB_REF=${MOVIE_NFO_LIB_REF}
ARG GIT_TOKEN
RUN if [ -z "$GIT_TOKEN" ]; then \
      echo "GIT_TOKEN is required to install movie_nfo_lib" >&2; exit 1; \
    fi \
    && TOKEN_URLENC=$(printf '%s' "$GIT_TOKEN" | sed 's/@/%40/g; s/:/%3A/g; s#/#%2F#g') \
    && python3 -m venv /opt/avior-venv \
    && /opt/avior-venv/bin/pip install --disable-pip-version-check --no-cache-dir --quiet -r /tmp/avior-requirements.txt \
    && /opt/avior-venv/bin/pip install --disable-pip-version-check --no-cache-dir --quiet "movie-metadata @ git+https://x-access-token:${TOKEN_URLENC}@github.com/bmaeofu/movie_nfo_lib@${MOVIE_NFO_LIB_REF}" \
    && rm -f /tmp/avior-requirements.txt \
    && unset TOKEN_URLENC GIT_TOKEN
ENV PATH="/opt/avior-venv/bin:$PATH"
COPY llm_compare /opt/avior-go-llm-compare

FROM linuxserver/ffmpeg:latest
COPY --from=build /out/avior-go /opt/avior-go-bin
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
