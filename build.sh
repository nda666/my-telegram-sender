#!/usr/bin/env bash
set -e

# Berhenti jika ada error
trap 'echo "❌ Build gagal!"; exit 1' ERR

USE_UPX=false
for arg in "$@"; do
  case $arg in
    --upx)
      USE_UPX=true
      ;;
  esac
done

echo "🚀 [1/3] Building frontend assets (Vite)..."
npm run build

echo "🔨 [2/3] Compiling Go binary (statically linked)..."
CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o telegram-sender .

if [ "$USE_UPX" = true ]; then
  if command -v upx >/dev/null 2>&1; then
    echo "🗜️  [Extra] Mengompres binary dengan UPX..."
    upx -q --lzma telegram-sender
  else
    echo "⚠️  Peringatan: 'upx' tidak ditemukan di sistem!"
    echo "    Silakan install dengan: sudo apt install upx (atau upx-ucl)"
    echo "    Melanjutkan proses tanpa UPX..."
  fi
fi

echo "📦 [3/3] Zipping telegram-sender binary..."
rm -f telegram-sender.zip
zip -q telegram-sender.zip telegram-sender

echo "✅ Selesai! Hasil build:"
ls -lh telegram-sender telegram-sender.zip
