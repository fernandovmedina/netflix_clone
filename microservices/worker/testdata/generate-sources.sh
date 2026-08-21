#!/bin/sh
set -eu
out="${1:-$(dirname "$0")/generated}"
mkdir -p "$out"
for height in 360 720 1080 1440; do
  width=$((height * 16 / 9 / 2 * 2))
  ffmpeg -hide_banner -loglevel error -y \
    -f lavfi -i "testsrc2=size=${width}x${height}:rate=24" \
    -f lavfi -i "sine=frequency=1000:sample_rate=48000" \
    -t 60 -c:v libx264 -preset veryfast -pix_fmt yuv420p -c:a aac \
    "$out/source-${height}p.mp4"
done
