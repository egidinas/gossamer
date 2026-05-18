#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
out_dir="${1:-/tmp/gossamer-brume2}"
archive="${out_dir}.tar.gz"

cd "$repo_root"

if [[ "${GOSSAMER_SKIP_FIXTURES:-0}" != "1" ]]; then
	go run ./cmd/gossamer-fixtures
fi

npm --prefix web run build

rm -rf "$out_dir" "$archive"
mkdir -p "$out_dir/fixtures" "$out_dir/web" "$out_dir/deploy/openwrt"

# This OpenWrt package is a legacy fallback. The canonical public origin should
# run on the Linux server with the normal, current Go toolchain; the router
# should only route/proxy. Keep the fallback artifact on the known-good router
# runtime unless GOSSAMER_BRUME2_GOTOOLCHAIN is explicitly set for a compatibility
# test. The fallback static-tile demo does not need the CGO-only DuckDB preview.
router_toolchain="${GOSSAMER_BRUME2_GOTOOLCHAIN:-go1.22.12}"
go_mod_backup="$(mktemp)"
cp go.mod "$go_mod_backup"
restore_go_mod() {
	cp "$go_mod_backup" go.mod
	rm -f "$go_mod_backup"
}
trap restore_go_mod EXIT
go mod edit -go=1.22
GOTOOLCHAIN="$router_toolchain" CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -tags noduckdb -trimpath -ldflags="-s -w" -o "$out_dir/gossamer-server" ./cmd/gossamer-server
restore_go_mod
trap - EXIT

cp -a fixtures/public "$out_dir/fixtures/public"
cp -a web/dist "$out_dir/web/dist"
cp deploy/openwrt/gossamer.init "$out_dir/gossamer.init"
cp -a deploy/openwrt "$out_dir/deploy/"

chmod +x "$out_dir/gossamer-server" "$out_dir/gossamer.init"
tar -C "$out_dir" -czf "$archive" .

du -h "$archive"
printf 'Package: %s\n' "$archive"
