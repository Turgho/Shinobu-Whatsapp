#!/bin/bash

# yt-dlp
wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp
chmod +x yt-dlp

# ffmpeg estático
wget https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
tar -xf ffmpeg-release-amd64-static.tar.xz
cp ffmpeg-*-static/ffmpeg .
rm -rf ffmpeg-*-static ffmpeg-release-amd64-static.tar.xz

# webpmux vem junto com o pacote webp
wget https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-1.5.0-linux-x86-64.tar.gz
tar -xf libwebp-1.5.0-linux-x86-64.tar.gz
cp libwebp-*/bin/webpmux .
rm -rf libwebp-* libwebp-1.5.0-linux-x86-64.tar.gz