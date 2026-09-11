FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY *.go ./
COPY web ./web
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/spotjspf .

FROM python:3.12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg ca-certificates wget \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir spotdl

COPY --from=builder /out/spotjspf /usr/local/bin/spotjspf

RUN useradd --create-home --uid 1000 appuser \
    && mkdir -p /downloads /data \
    && chown -R appuser:appuser /downloads /data

USER appuser
WORKDIR /home/appuser

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/spotjspf"]
