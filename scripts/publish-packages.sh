#!/bin/sh
# Fork-only helper: builds Entware packages and lays them out as opkg feeds for the `packages` branch.
#
# Usage:
#   scripts/publish-packages.sh <out_dir> [config/entware/<target>.config ...]
#
# Result, one feed per target:
#   <out_dir>/entware/<target>/magitrickle.ipk   stable name, direct `opkg install <url>`
#   <out_dir>/entware/<target>/Packages[.gz]      index, feed for `opkg update && opkg upgrade`
#   <out_dir>/README.md
#
# Environment:
#   BASE_URL  raw URL of the packages branch root, used in README.md
set -eu

OUT="$1"
shift
[ $# -gt 0 ] || set -- config/entware/*.config
BASE_URL="${BASE_URL:-https://raw.githubusercontent.com/freeman1111/MagiTrickle/packages}"

cd "$(git rev-parse --show-toplevel)"
eval export $(make _return_export_dynamic_env)

targets=""
for cfg in "$@"; do
  PLATFORM=$(sed -n 's/^PLATFORM=//p' "$cfg")
  TARGET=$(sed -n 's/^TARGET=//p' "$cfg")
  if [ "$PLATFORM" != "entware" ]; then
    echo ">> skipping $cfg: only entware feeds are published" >&2
    continue
  fi

  echo ">> building $PLATFORM/$TARGET"
  cp "$cfg" .config
  rm -f .build/magitrickle_*_"${PLATFORM}_${TARGET}".ipk
  make
  ipk=$(ls .build/magitrickle_*_"${PLATFORM}_${TARGET}".ipk)

  dir="$OUT/$PLATFORM/$TARGET"
  mkdir -p "$dir"
  rm -f "$dir"/*.ipk "$dir"/Packages "$dir"/Packages.gz
  cp "$ipk" "$dir/magitrickle.ipk"

  # ipk here is a tar.gz with control.tar.gz inside (see Makefile)
  {
    tar -xzOf "$dir/magitrickle.ipk" ./control.tar.gz | tar -xzO ./control | sed '/^$/d'
    echo "Filename: magitrickle.ipk"
    echo "Size: $(wc -c < "$dir/magitrickle.ipk" | tr -d ' ')"
    echo "SHA256sum: $(sha256sum "$dir/magitrickle.ipk" | cut -d' ' -f1)"
    echo
  } > "$dir/Packages"
  gzip -9nc "$dir/Packages" > "$dir/Packages.gz"

  targets="$targets $TARGET"
done

{
  echo "# MagiTrickle fork packages"
  echo
  echo "Built from $(git rev-parse --short HEAD). Install or update with one command:"
  echo
  echo '```sh'
  for t in $targets; do
    echo "# $t"
    echo "opkg install $BASE_URL/entware/$t/magitrickle.ipk"
  done
  echo '```'
} > "$OUT/README.md"

echo ">> feeds written to $OUT"
