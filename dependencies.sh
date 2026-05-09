#!/bin/bash
set -e

mkdir -p bin

# yt-dlp
curl -L -o bin/yt-dlp \
https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux
chmod +x bin/yt-dlp

# ffmpeg estático
wget -q https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
tar -xf ffmpeg-release-amd64-static.tar.xz

cp ffmpeg-*-static/ffmpeg bin/
chmod +x bin/ffmpeg

rm -rf ffmpeg-*-static ffmpeg-release-amd64-static.tar.xz

# webpmux
wget -q https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-1.5.0-linux-x86-64.tar.gz
tar -xf libwebp-1.5.0-linux-x86-64.tar.gz

cp libwebp-*/bin/webpmux bin/
chmod +x bin/webpmux

rm -rf libwebp-* libwebp-1.5.0-linux-x86-64.tar.gz