#!/usr/bin/env bash
set -e

# Script de instalação das ferramentas de download e processamento de vídeo (yt-dlp e ffmpeg)
# Suporta distribuições Debian/Ubuntu (incluindo Ubuntu 20.04, 22.04 e 24.04).

echo "=========================================================="
echo " Presenter — Instalador de Ferramentas de Vídeo"
echo " (yt-dlp standalone com Python embutido + ffmpeg)"
echo "=========================================================="

if [ "$EUID" -ne 0 ]; then
  echo ""
  echo "Erro: Este script precisa ser executado como root ou com sudo."
  echo "Uso: sudo ./install-video-tools.sh"
  exit 1
fi

echo ""
echo "[1/4] Atualizando repositórios de pacotes..."
apt-get update -y

echo ""
echo "[2/4] Instalando dependências do sistema (ffmpeg, wget, ca-certificates)..."
apt-get install -y ffmpeg wget curl ca-certificates

echo ""
echo "[3/4] Baixando binário compilado independente do yt-dlp (yt-dlp_linux)..."
# Baixa o binário ELF oficial independente, que traz seu próprio runtime Python embutido
wget -q --show-progress "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" -O /usr/local/bin/yt-dlp
chmod a+rx /usr/local/bin/yt-dlp

echo ""
echo "[4/4] Verificando instalação das ferramentas..."
if command -v yt-dlp >/dev/null 2>&1; then
    YTDLP_VER=$(yt-dlp --version 2>/dev/null || echo "ok")
    echo "  ✓ yt-dlp instalado com sucesso! Versão: ${YTDLP_VER}"
else
    echo "  ✗ Falha ao verificar yt-dlp."
    exit 1
fi

if command -v ffmpeg >/dev/null 2>&1; then
    FFMPEG_VER=$(ffmpeg -version 2>&1 | head -n 1)
    echo "  ✓ ffmpeg instalado com sucesso! (${FFMPEG_VER})"
else
    echo "  ✗ Falha ao verificar ffmpeg."
    exit 1
fi

echo ""
echo "=========================================================="
echo " Tudo pronto! As ferramentas de vídeo foram instaladas."
echo " O Presenter agora pode baixar e mesclar vídeos do YouTube."
echo "=========================================================="
