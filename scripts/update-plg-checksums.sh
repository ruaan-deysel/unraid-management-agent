#!/bin/bash
# Writes the release bundle's MD5 and SHA256 into the .plg entities and checks
# the result (#247). Used by the release workflow before publishing and again on
# main before the checksum commit.
#
# Usage: scripts/update-plg-checksums.sh <md5> <sha256> <version> [plg]
#   <version> must equal the .plg's "version" entity, so checksums are never
#   written into a manifest that points at a different bundle.

set -euo pipefail

if [ $# -lt 3 ] || [ $# -gt 4 ]; then
    echo "Usage: $0 <md5> <sha256> <version> [plg]" >&2
    exit 2
fi
md5="$1"
sha256="$2"
version="$3"
plg="${4:-unraid-management-agent.plg}"

# Prints an error and exits non-zero.
fail() {
    echo "✗ $*" >&2
    exit 1
}

# Prints the value of entity $1 in the .plg, or nothing if it is missing.
entity() {
    sed -n -E "s/^[[:space:]]*<!ENTITY[[:space:]]+$1[[:space:]]+\"([^\"]*)\">.*/\1/p" "$plg" | head -n1
}

[ -f "$plg" ] || fail "PLG not found: $plg"
[[ "$md5" =~ ^[0-9a-f]{32}$ ]] || fail "MD5 is not 32 lowercase hex characters: '$md5'"
[[ "$sha256" =~ ^[0-9a-f]{64}$ ]] || fail "SHA256 is not 64 lowercase hex characters: '$sha256'"

plg_version=$(entity version)
[ "$plg_version" = "$version" ] ||
    fail "PLG version '$plg_version' does not match release version '$version'; not writing checksums"

# Replace only the quoted value so trailing comments (the detect-secrets
# pragmas) survive. Write through a temp file: BSD and GNU sed differ on -i.
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
sed -E \
    -e "s/^([[:space:]]*<!ENTITY[[:space:]]+md5[[:space:]]+\")[^\"]*(\">)/\1${md5}\2/" \
    -e "s/^([[:space:]]*<!ENTITY[[:space:]]+sha256[[:space:]]+\")[^\"]*(\">)/\1${sha256}\2/" \
    "$plg" >"$tmp"
cat "$tmp" >"$plg"

[ "$(entity md5)" = "$md5" ] || fail "md5 entity in $plg is '$(entity md5)', expected '$md5'"
[ "$(entity sha256)" = "$sha256" ] || fail "sha256 entity in $plg is '$(entity sha256)', expected '$sha256'"
echo "✓ $plg pins v${version}: MD5 ${md5}, SHA256 ${sha256}"
