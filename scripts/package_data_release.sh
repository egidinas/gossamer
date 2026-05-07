#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
DATA_VERSION=${DATA_VERSION:-current}
OUT_DIR=${GOSSAMER_RELEASE_DIR:-$ROOT/dist/releases}
DATA_DIR="$ROOT/fixtures/public_tiles/$DATA_VERSION"

if [ ! -d "$DATA_DIR" ]; then
  echo "missing tile data bundle: $DATA_DIR" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
tar -C "$DATA_DIR" -czf "$OUT_DIR/gossamer-data-$DATA_VERSION.tgz" .

du -sh "$DATA_DIR"
du -h "$OUT_DIR/gossamer-data-$DATA_VERSION.tgz"
find "$DATA_DIR" -type f | wc -l
printf '%s\n' "$OUT_DIR/gossamer-data-$DATA_VERSION.tgz"
