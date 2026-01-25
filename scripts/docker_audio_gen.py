#!/usr/bin/env python3

import argparse
import json
import os
import subprocess
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--force", action="store_true", help="Force regeneration of existing files")
args = parser.parse_args()

VOICE = os.environ["VOICE"]
VOICES_DIR = os.environ["VOICES_DIR"]
ROUTES_FILE = os.environ.get("ROUTES_FILE", "/audio/routes.json")
OUT = os.environ.get("AUDIO_OUT_DIR", "out")
os.makedirs(OUT, exist_ok=True)

with open(ROUTES_FILE) as f:
    devices = json.load(f)

commands = []
for device_name, device_config in devices.items():
    os.makedirs(f"{OUT}/{device_name}", exist_ok=True)
    for audio_name, command_info in device_config["commands"].items():
        commands.append({
            "device": device_name,
            "audio_name": audio_name,
            "text": command_info["text"]
        })

print(f"Processing {len(commands)} commands")

generated = 0
skipped = 0

for i, cmd in enumerate(commands, 1):
    device = cmd["device"]
    audio_name = cmd["audio_name"]
    text = cmd["text"]

    filename = f"{OUT}/{device}/{audio_name}.wav"

    if os.path.exists(filename) and not args.force:
        print(f"[{i}/{len(commands)}] Skipping (exists): {audio_name}.wav")
        skipped += 1
        continue

    print(f"[{i}/{len(commands)}] Generating: {audio_name}")

    subprocess.run(
        [
            sys.executable, "-m", "piper",
            "-m", VOICE,
            "-f", filename,
            "--data-dir", VOICES_DIR,
            "--", text,
        ],
        check=True,
    )
    print(f"[{i}/{len(commands)}] Generated: {audio_name}.wav")
    generated += 1

print(f"Generated {generated}, skipped {skipped} (total: {len(commands)})")
