#!/bin/bash
set -e
mkdir -p bin

# yt-dlp (já pega latest)
echo "Instalando yt-dlp..."
curl -L -o bin/yt-dlp \
    https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux
chmod +x bin/yt-dlp

# ffmpeg estático (johnvansickle sempre serve o latest)
echo "Instalando ffmpeg..."
wget -q https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
tar -xf ffmpeg-release-amd64-static.tar.xz
cp ffmpeg-*-static/ffmpeg bin/
chmod +x bin/ffmpeg
rm -rf ffmpeg-*-static ffmpeg-release-amd64-static.tar.xz

# webpmux (resolve latest via GitHub API)
echo "Instalando webpmux..."
WEBP_VERSION=$(curl -s https://api.github.com/repos/webmproject/libwebp/tags \
    | grep '"name"' | head -1 | grep -oP '(?<=v)[0-9]+\.[0-9]+\.[0-9]+')
WEBP_URL="https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-${WEBP_VERSION}-linux-x86-64.tar.gz"
echo "Versão libwebp: ${WEBP_VERSION}"
wget -q "$WEBP_URL"
tar -xf "libwebp-${WEBP_VERSION}-linux-x86-64.tar.gz"
cp "libwebp-${WEBP_VERSION}-linux-x86-64/bin/webpmux" bin/
chmod +x bin/webpmux
rm -rf "libwebp-${WEBP_VERSION}-linux-x86-64" "libwebp-${WEBP_VERSION}-linux-x86-64.tar.gz"

echo "Concluído:"
bin/yt-dlp --version
bin/ffmpeg -version | head -1
bin/webpmux -version