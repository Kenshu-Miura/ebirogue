#!/usr/bin/env bash
# build_web.sh - WASM ビルドとブラウザ配布用ファイルを public/ に生成する。
# Netlify のビルドコマンド、およびローカルでのブラウザ動作確認の両方で使う。
set -euo pipefail

cd "$(dirname "$0")"

OUT_DIR="public"
mkdir -p "$OUT_DIR"

echo "==> WASM をビルドします"
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUT_DIR/ebirogue.wasm" .

echo "==> wasm_exec.js を Go インストールからコピーします"
GOROOT="$(go env GOROOT)"
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
  # Go 1.24 以降
  cp "$GOROOT/lib/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
elif [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
  # Go 1.23 以前
  cp "$GOROOT/misc/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
else
  echo "wasm_exec.js が見つかりません（GOROOT=$GOROOT）" >&2
  exit 1
fi

echo "==> 静的ファイルをコピーします"
cp web/index.html "$OUT_DIR/index.html"

echo "==> 完了: $OUT_DIR/ に index.html, ebirogue.wasm, wasm_exec.js を生成しました"
