FROM golang:1.25-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/kids-planet-be ./cmd/server

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates ffmpeg tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 10001 appuser

WORKDIR /app
COPY --from=builder /out/kids-planet-be /app/kids-planet-be
COPY etc/backend.docker.yaml /app/etc/backend.yaml
RUN mkdir -p /data/resources/song /data/resources/video /data/resources/lyrics /data/resources/poster /data/generated \
    && chown -R appuser:appuser /app /data

USER appuser
EXPOSE 8888

ENTRYPOINT ["/app/kids-planet-be"]
CMD ["-f", "/app/etc/backend.yaml"]
