#!/bin/sh
set -eu

ROOT=${GOSSAMER_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)}
BRUME2_HOST=${BRUME2_HOST:-root@192.168.8.1}
REMOTE_ROOT=${GOSSAMER_REMOTE_ROOT:-/opt/gossamer}
DATA_VERSION=${DATA_VERSION:-current}
ARCHIVE=${1:-$ROOT/dist/releases/gossamer-data-$DATA_VERSION.tgz}
REMOTE_TMP="/tmp/gossamer-data-$DATA_VERSION.tgz"

if [ ! -f "$ARCHIVE" ]; then
  echo "missing data archive: $ARCHIVE" >&2
  exit 1
fi

ssh "$BRUME2_HOST" "df -h /tmp '$REMOTE_ROOT' 2>/dev/null || df -h /tmp"
scp "$ARCHIVE" "$BRUME2_HOST:$REMOTE_TMP"
ssh "$BRUME2_HOST" "set -eu
mkdir -p '$REMOTE_ROOT/data/releases/$DATA_VERSION' '$REMOTE_ROOT/fixtures/public_tiles'
tar -xzf '$REMOTE_TMP' -C '$REMOTE_ROOT/data/releases/$DATA_VERSION'
rm -f '$REMOTE_ROOT/data/current.next'
ln -s 'releases/$DATA_VERSION' '$REMOTE_ROOT/data/current.next'
rm -f '$REMOTE_ROOT/data/current'
mv '$REMOTE_ROOT/data/current.next' '$REMOTE_ROOT/data/current'
rm -rf '$REMOTE_ROOT/fixtures/public_tiles/current'
ln -s '$REMOTE_ROOT/data/current' '$REMOTE_ROOT/fixtures/public_tiles/current'
rm -f '$REMOTE_TMP'
wget -T 8 -t 1 -qO- http://[::1]:8095/data/current/manifest.json >/dev/null
df -h /tmp '$REMOTE_ROOT' 2>/dev/null || df -h /tmp
"
printf 'deployed data %s to %s\n' "$DATA_VERSION" "$BRUME2_HOST"
