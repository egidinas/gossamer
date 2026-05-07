#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
UI_VERSION=${UI_VERSION:-$(date -u +%Y%m%dT%H%M%SZ)}
DATA_VERSION=${DATA_VERSION:-$(date -u +%Y%m%dT%H%M%SZ)}

DATA_VERSION=$DATA_VERSION "$ROOT/scripts/build_tile_bundle.sh"
UI_VERSION=$UI_VERSION "$ROOT/scripts/package_ui_release.sh"
DATA_VERSION=$DATA_VERSION "$ROOT/scripts/package_data_release.sh"
UI_VERSION=$UI_VERSION "$ROOT/scripts/deploy_brume2_ui.sh"
DATA_VERSION=$DATA_VERSION "$ROOT/scripts/deploy_brume2_data.sh"
