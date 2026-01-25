# jacadi

jacadi is a simple Go HTTP API server that plays WAV audio files through a USB speaker. It is designed to run on a Raspberry Pi or similar with a USB speaker attached, and allow clients to play sounds via HTTP requests.

jacadi's main ambition is to serve as a voice proxy to command voice controlled devices (like alexa, ok google, etc).


## Quick Start

### 1. Verify USB Speaker

On your Raspberry Pi, check that the USB speaker is detected:

```bash
aplay -l
```

You should see your USB speaker listed. Note the card number.

### 2. Build and Run

```bash
# Build the Docker image (slim - pre-generated audio only)
docker build --target slim -t jacadi:slim .

# Or build full image with TTS support
docker build --target full -t jacadi:full .

# Build with a specific routeset (defaults to dreame)
docker build --target slim --build-arg ROUTES=mydevice -t jacadi:mydevice .

# Start the service
docker-compose up -d
```

### 3. Test

```bash
# Health check
curl http://localhost:8080/health

# Play audio (format: /play/{device}/{command})
curl -X POST http://localhost:8080/play/dreame/ok-dream
curl -X POST http://localhost:8080/play/dreame/clean-kitchen
```

The full image embeds piper, allowing on the fly TTS through the API:

```bash
curl -X POST http://localhost:8080/play/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world"}'

curl -X POST http://localhost:8080/play/tts \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world", "voice": "en_US-amy-low"}'
```

## Configuration

### Environment Variables

- `EXTRA_ROUTES_PATH`: Path to extra routes file for runtime merging (optional)
- `HOST`: Listen address (default: `0.0.0.0`)
- `PORT`: Listen port (default: `8080`)
- `AUDIODEV`: ALSA device for audio output (e.g., `hw:3,0`)
- `VOICE`: Default piper voice model (default: `en_US-amy-low`)

### Route Files

Routes are stored in `routes/{name}.json`. Use the `ROUTES` build arg to select a routeset (defaults to `dreame`).

```json
{
  "dreame": {
    "commands": {
      "ok-dream": { "text": "Okay dream" },
      "clean-kitchen": { "text": "Clean the kitchen" }
    }
  }
}
```

- Top-level keys are device names (creates `/play/{device}/...` endpoints)
- `commands`: Map of command names to metadata
- Audio files go in `assets/audio/{device}/{command}.wav` (copied to `/audio/` at build time)

### Extra Routes

Add routes at runtime without rebuilding by mounting an `extra_routes.json` file:

```yaml
volumes:
  - "./extra_routes.json:/app/extra_routes.json:ro"
  - "./extra_audio:/audio/extra"
environment:
  - EXTRA_ROUTES_PATH=/app/extra_routes.json
```

Audio files go in `/audio/extra/{device}/{command}.wav`.

- **Full image**: Missing audio files are auto-generated at startup using piper TTS
- **Slim image**: You must provide the audio files manually

In docker, ensure `/audio/extra` is a volume so the generated audio files persist.

### Custom Audio Files

Audio files must be WAV format: 44100 Hz, 16-bit, mono.

Convert with ffmpeg:

```bash
ffmpeg -i input.wav -ar 44100 -ac 1 -acodec pcm_s16le output.wav
```

## Extras

### Home Assistant Config Generator

Generate `rest_command` configuration for Home Assistant from your routes:

```bash
go run cmd/generate-homeassistant/main.go -base-url="http://jacadi.local:8080"
```

With TTS support (full image only):

```bash
go run cmd/generate-homeassistant/main.go -base-url="http://jacadi.local:8080" -tts
```

Output goes to `ha-config/`. Include in your Home Assistant `configuration.yaml`:

```yaml
rest_command: !include homeassistant_rest.yml
script: !include homeassistant_scripts.yml
```
