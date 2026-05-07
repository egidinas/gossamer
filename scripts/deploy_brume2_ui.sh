#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
BRUME2_HOST=${BRUME2_HOST:-root@192.168.8.1}
REMOTE_ROOT=${GOSSAMER_REMOTE_ROOT:-/opt/gossamer}
UI_VERSION=${UI_VERSION:-$(date -u +%Y%m%dT%H%M%SZ)}
ARCHIVE=${1:-$ROOT/dist/releases/gossamer-ui-$UI_VERSION.tgz}
REMOTE_TMP="/tmp/gossamer-ui-$UI_VERSION.tgz"

if [ ! -f "$ARCHIVE" ]; then
  echo "missing UI archive: $ARCHIVE" >&2
  exit 1
fi

ssh "$BRUME2_HOST" "df -h /tmp '$REMOTE_ROOT' 2>/dev/null || df -h /tmp"
scp "$ARCHIVE" "$BRUME2_HOST:$REMOTE_TMP"
ssh "$BRUME2_HOST" "set -eu
mkdir -p '$REMOTE_ROOT/ui/releases/$UI_VERSION' '$REMOTE_ROOT/web' '$REMOTE_ROOT/fixtures/public_tiles'
tar -xzf '$REMOTE_TMP' -C '$REMOTE_ROOT/ui/releases/$UI_VERSION'
rm -f '$REMOTE_ROOT/ui/current.next'
ln -s 'releases/$UI_VERSION' '$REMOTE_ROOT/ui/current.next'
rm -f '$REMOTE_ROOT/ui/current'
mv '$REMOTE_ROOT/ui/current.next' '$REMOTE_ROOT/ui/current'
rm -rf '$REMOTE_ROOT/web/dist'
ln -s '$REMOTE_ROOT/ui/current' '$REMOTE_ROOT/web/dist'
rm -f '$REMOTE_TMP'
wget -T 8 -t 1 -qO- http://[::1]:8095/healthz >/dev/null
df -h /tmp '$REMOTE_ROOT' 2>/dev/null || df -h /tmp
"
printf 'deployed UI %s to %s\n' "$UI_VERSION" "$BRUME2_HOST"
