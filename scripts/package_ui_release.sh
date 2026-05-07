#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
UI_VERSION=${UI_VERSION:-$(date -u +%Y%m%dT%H%M%SZ)}
OUT_DIR=${GOSSAMER_RELEASE_DIR:-$ROOT/dist/releases}

cd "$ROOT"
npm --prefix web run build
mkdir -p "$OUT_DIR"
printf '%s\n' "$UI_VERSION" > web/dist/ui-version.txt
printf 'ok\n' > web/dist/healthz
tar -C web/dist -czf "$OUT_DIR/gossamer-ui-$UI_VERSION.tgz" .

du -h "$OUT_DIR/gossamer-ui-$UI_VERSION.tgz"
printf '%s\n' "$OUT_DIR/gossamer-ui-$UI_VERSION.tgz"
