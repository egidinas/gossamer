#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
DATA_DIR=${DATA_DIR:-$ROOT/fixtures/public_tiles/current}
UI_DIR=${UI_DIR:-$ROOT/web/dist}
MAX_DATA_KB=${MAX_DATA_KB:-8192}
MAX_TILE_KB=${MAX_TILE_KB:-512}
MAX_TILE_FILES=${MAX_TILE_FILES:-200}
MAX_UI_JS_KB=${MAX_UI_JS_KB:-2048}

data_kb=$(du -sk "$DATA_DIR" | awk '{print $1}')
tile_files=$(find "$DATA_DIR" -type f | wc -l)
largest_tile_kb=$(find "$DATA_DIR" -type f -name '*.json' -exec du -k {} + | awk 'max<$1{max=$1} END{print max+0}')
ui_js_kb=0
if [ -d "$UI_DIR/assets" ]; then
  ui_js_kb=$(find "$UI_DIR/assets" -type f -name '*.js' -exec du -k {} + | awk '{sum+=$1} END{print sum+0}')
fi

printf 'static budget: data=%sKB files=%s largest_tile=%sKB ui_js=%sKB\n' "$data_kb" "$tile_files" "$largest_tile_kb" "$ui_js_kb"
test "$data_kb" -le "$MAX_DATA_KB"
test "$tile_files" -le "$MAX_TILE_FILES"
test "$largest_tile_kb" -le "$MAX_TILE_KB"
test "$ui_js_kb" -le "$MAX_UI_JS_KB"
