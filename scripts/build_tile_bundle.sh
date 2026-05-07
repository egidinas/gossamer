#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
DATA_VERSION=${DATA_VERSION:-$(date -u +%Y%m%dT%H%M%SZ)}
LEVELS=${GOSSAMER_TILE_LEVELS:-minute}
CAMPAIGNS=${GOSSAMER_CAMPAIGNS:-thermal_acceptance_fat,tvac_qualification}

cd "$ROOT"
go run ./cmd/gossamer-tiles \
  -data-version "$DATA_VERSION" \
  -levels "$LEVELS" \
  -campaigns "$CAMPAIGNS"

du -sh "fixtures/public_tiles/$DATA_VERSION" "fixtures/public_tiles/current"
find "fixtures/public_tiles/$DATA_VERSION" -type f | wc -l
