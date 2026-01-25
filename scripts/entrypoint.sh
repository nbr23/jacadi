#! /bin/bash

if [ "$PIPER_EMBEDDED" = "true" ]; then
    echo "Checking for missing audio files..."
    ROUTES_FILE=/app/routes.json AUDIO_OUT_DIR=/audio python /app/scripts/docker_audio_gen.py || \
        echo "Warning: Audio generation failed for routes.json, continuing anyway"

    if [ -n "$EXTRA_ROUTES_PATH" ] && [ -f "$EXTRA_ROUTES_PATH" ]; then
        echo "Generating audio for extra routes..."
        ROUTES_FILE="$EXTRA_ROUTES_PATH" AUDIO_OUT_DIR=/audio/extra python /app/scripts/docker_audio_gen.py || \
            echo "Warning: Audio generation failed for extra routes, continuing anyway"
    fi
fi

exec ./jacadi
