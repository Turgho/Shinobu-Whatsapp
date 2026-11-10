#!/bin/bash
set -e
mkdir -p bin

# Versões fixas: com "latest" cada setup podia trazer um binário
# diferente do que foi testado (e deno + yt-dlp precisam combinar).
YTDLP_VERSION="2025.09.25"
DENO_VERSION="2.4.3"

echo "Instalando yt-dlp ${YTDLP_VERSION}..."
curl -L -o bin/yt-dlp \
    "https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp_linux"
chmod +x bin/yt-dlp

# Deno — runtime JS obrigatório para os desafios EJS do YouTube (yt-dlp-ejs).
# O binário oficial yt-dlp_linux já embute os scripts; só o runtime é exigido.
# Mínimo suportado pelo yt-dlp: deno >= 2.3.0.
echo "Instalando deno..."
if ! command -v unzip >/dev/null 2>&1; then
    echo "ERRO: 'unzip' ausente (instale: sudo pacman -S unzip)" >&2
    exit 1
fi
curl -fsSL -o /tmp/deno.zip \
    "https://github.com/denoland/deno/releases/download/v${DENO_VERSION}/deno-x86_64-unknown-linux-gnu.zip"
unzip -p /tmp/deno.zip > bin/deno
chmod +x bin/deno
rm -f /tmp/deno.zip

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