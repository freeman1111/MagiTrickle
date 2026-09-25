#!/bin/sh
# Fork-only helper: builds Entware packages and lays them out as opkg feeds for the `packages` branch.
#
# Usage:
#   scripts/publish-packages.sh <out_dir> [config/entware/<target>.config ...]
#
# Result, one feed per target:
#   <out_dir>/entware/<target>/magitrickle.ipk                      latest build, stable link for `opkg install <url>`
#   <out_dir>/entware/<target>/magitrickle_<version>_<target>.ipk  the same build with its version in the name
#   <out_dir>/entware/<target>/Packages[.gz]                       index, feed for `opkg update && opkg upgrade`
#   <out_dir>/README.md
#
# Packages are marked as a fork: maintainer, URL and description point to the fork, not to the original
# authors (GPL: users of a derived work must be able to tell it apart from the original).
#
# Environment:
#   BASE_URL    raw URL of the packages branch root, used in README.md
#   FORK_REPO   fork repository URL
set -eu

OUT="$1"
shift
[ $# -gt 0 ] || set -- config/entware/*.config
BASE_URL="${BASE_URL:-https://raw.githubusercontent.com/freeman1111/MagiTrickle/packages}"
FORK_REPO="${FORK_REPO:-https://github.com/freeman1111/MagiTrickle}"
UPSTREAM_REPO="https://gitlab.com/magitrickle/magitrickle"

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
  make \
    PKG_URL="$FORK_REPO" \
    PKG_MAINTAINER="freeman1111 <$FORK_REPO/issues>" \
    PKG_DESCRIPTION="DNS-based routing application (unofficial fork of MagiTrickle, $UPSTREAM_REPO)"
  ipk=$(ls .build/magitrickle_*_"${PLATFORM}_${TARGET}".ipk)

  dir="$OUT/$PLATFORM/$TARGET"
  mkdir -p "$dir"
  rm -f "$dir"/*.ipk "$dir"/Packages "$dir"/Packages.gz
  version=$(tar -xzOf "$ipk" ./control.tar.gz | tar -xzO ./control | sed -n 's/^Version: //p')
  versioned="magitrickle_${version}_${TARGET}.ipk"
  cp "$ipk" "$dir/magitrickle.ipk"
  cp "$ipk" "$dir/$versioned"

  # ipk here is a tar.gz with control.tar.gz inside (see Makefile)
  {
    tar -xzOf "$dir/magitrickle.ipk" ./control.tar.gz | tar -xzO ./control | sed '/^$/d'
    echo "Filename: $versioned"
    echo "Size: $(wc -c < "$dir/magitrickle.ipk" | tr -d ' ')"
    echo "SHA256sum: $(sha256sum "$dir/magitrickle.ipk" | cut -d' ' -f1)"
    echo
  } > "$dir/Packages"
  gzip -9nc "$dir/Packages" > "$dir/Packages.gz"

  targets="$targets $TARGET:$versioned"
done

commit=$(git rev-parse HEAD)
{
  echo "# MagiTrickle fork packages"
  echo
  echo "Unofficial fork of [MagiTrickle]($UPSTREAM_REPO) by Vladimir Avtsenov and contributors,"
  echo "licensed under GPL-3.0-or-later. Report problems with these builds to the fork: $FORK_REPO/issues,"
  echo "not to the original authors."
  echo
  echo "Source code of these builds: $FORK_REPO/tree/$commit"
  echo
  echo "Latest build, the link never changes:"
  echo
  echo '```sh'
  for entry in $targets; do
    echo "# ${entry%%:*}"
    echo "opkg install $BASE_URL/entware/${entry%%:*}/magitrickle.ipk"
  done
  echo '```'
  echo
  echo "The same builds with the version in the file name:"
  echo
  echo '```sh'
  for entry in $targets; do
    echo "opkg install $BASE_URL/entware/${entry%%:*}/${entry#*:}"
  done
  echo '```'
} > "$OUT/README.md"

echo ">> feeds written to $OUT"
