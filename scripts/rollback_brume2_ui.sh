#!/bin/sh
set -eu

BRUME2_HOST=${BRUME2_HOST:-}
REMOTE_ROOT=${GOSSAMER_REMOTE_ROOT:-/opt/gossamer}
TARGET_VERSION=${1:-}

if [ -z "$BRUME2_HOST" ]; then
  echo "set BRUME2_HOST to a local SSH host alias or user@host target" >&2
  exit 2
fi

ssh "$BRUME2_HOST" "set -eu
if [ -z '$TARGET_VERSION' ]; then
  target=\$(ls -1t '$REMOTE_ROOT/ui/releases' | sed -n '2p')
else
  target='$TARGET_VERSION'
fi
test -n \"\$target\"
test -d '$REMOTE_ROOT/ui/releases/'\"\$target\"
rm -f '$REMOTE_ROOT/ui/current.next'
ln -s 'releases/'\"\$target\" '$REMOTE_ROOT/ui/current.next'
rm -f '$REMOTE_ROOT/ui/current'
mv '$REMOTE_ROOT/ui/current.next' '$REMOTE_ROOT/ui/current'
rm -rf '$REMOTE_ROOT/web/dist'
ln -s '$REMOTE_ROOT/ui/current' '$REMOTE_ROOT/web/dist'
wget -T 8 -t 1 -qO- http://[::1]:8095/healthz >/dev/null
printf 'rolled back UI to %s\n' \"\$target\"
"
