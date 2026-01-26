# Audio files generator
FROM --platform=${BUILDOS}/${BUILDARCH} python:3.12-slim AS audiogen

ARG ROUTES="dreame"
ARG VOICE="en_US-amy-low"
ENV VOICE=$VOICE
ENV VOICES_DIR="/voices"

RUN useradd --create-home --shell /bin/bash python
RUN mkdir -p /audio && chown python /audio

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/
RUN uv venv /opt/venv && chmod -R a+rX /opt/venv
ENV VIRTUAL_ENV=/opt/venv
ENV PATH="/opt/venv/bin:$PATH"

RUN uv pip install piper-tts
RUN mkdir -p $VOICES_DIR && chown python $VOICES_DIR && chmod 755 $VOICES_DIR

USER python

RUN python3 -m piper.download_voices --download_dir $VOICES_DIR $VOICE

WORKDIR /audio

COPY ./scripts/docker_audio_gen.py .
COPY ./routes/${ROUTES}.json ./routes.json

RUN ./docker_audio_gen.py

# Go app builder (Alpine/musl for slim image)
FROM --platform=${BUILDOS}/${BUILDARCH} golang:1.24-alpine AS builder-alpine

RUN apk add --no-cache gcc musl-dev alsa-lib-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go go.* .
COPY audio ./audio
COPY handlers ./handlers
COPY config ./config
COPY tts ./tts

RUN GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o jacadi .

# Go app builder (Debian/glibc for full image)
FROM golang:1.24 AS builder-debian

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libasound2-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY *.go go.* .
COPY audio ./audio
COPY handlers ./handlers
COPY config ./config
COPY tts ./tts

RUN go build -ldflags="-s -w" -o jacadi .

# Slim Runtime (pre generated audio only)
FROM alpine:latest AS slim

ARG ROUTES="dreame"
ENV AUDIO_BASE_PATH="/audio"
ENV PORT="8080"
ENV AUDIODEV=""

RUN apk add --no-cache alsa-lib alsa-utils pulseaudio-utils ca-certificates
RUN adduser -S go -G audio

WORKDIR /app

COPY --from=builder-alpine /build/jacadi .
COPY ./routes/${ROUTES}.json ./routes.json
COPY --from=audiogen /audio/out/ /audio

USER go

EXPOSE ${PORT}

CMD ["./jacadi"]

# Full runtime with piper
FROM python:3.12-slim AS full

ARG ROUTES="dreame"
ARG VOICE="en_US-amy-low"
ENV AUDIO_BASE_PATH="/audio"
ENV PORT="8080"
ENV AUDIODEV=""
ENV PIPER_EMBEDDED="true"
ENV PIPER_SAMPLE_RATE="16000"
ENV VOICE=$VOICE
ENV VOICES_DIR="/voices"

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/
RUN uv venv /opt/venv && chmod -R a+rX /opt/venv
ENV VIRTUAL_ENV=/opt/venv
ENV PATH="/opt/venv/bin:$PATH"

RUN apt update && apt install --no-install-recommends -y \
    alsa-utils \
    libasound2 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -r -g audio -G audio -m -s /bin/bash appuser
RUN mkdir -p /audio /audio/extra && chown -R appuser:audio /audio

RUN uv pip install piper-tts
# RUN mkdir -p $VOICES_DIR && chown -R appuser $VOICES_DIR && chmod 755 $VOICES_DIR

USER appuser

# RUN python3 -m piper.download_voices --download_dir $VOICES_DIR $VOICE


WORKDIR /app

COPY --chown=appuser:audio --from=builder-debian /build/jacadi .
COPY --chown=appuser:audio ./routes/${ROUTES}.json ./routes.json
COPY --chown=appuser:audio ./scripts/docker_audio_gen.py ./scripts/
COPY --chown=appuser:audio ./scripts/entrypoint.sh ./scripts/
COPY --chown=appuser:audio --from=audiogen /audio/out/ /audio
COPY --chown=appuser:audio --from=audiogen /voices $VOICES_DIR

EXPOSE ${PORT}

CMD ["./scripts/entrypoint.sh"]
