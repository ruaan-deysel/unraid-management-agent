#!/bin/bash
# Tests for scripts/update-plg-checksums.sh (#247), run by `make test-packaging`.
#
# Usage: bash scripts/test-plg-checksums.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UPDATE="${REPO_ROOT}/scripts/update-plg-checksums.sh"
PLG="${REPO_ROOT}/unraid-management-agent.plg"

# Prints an error and exits non-zero.
fail() {
    echo "✗ $*" >&2
    exit 1
}

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM

version=$(sed -n -E 's/^[[:space:]]*<!ENTITY[[:space:]]+version[[:space:]]+"([^"]*)">.*/\1/p' "$PLG")
md5=$(printf '%032d' 1 | tr 0 a)
sha256=$(printf '%064d' 2 | tr 0 b)

cp "$PLG" "${work}/ok.plg"
bash "$UPDATE" "$md5" "$sha256" "$version" "${work}/ok.plg" >/dev/null || fail "update failed on a valid input"
grep -q "<!ENTITY md5         \"${md5}\"> <!-- pragma: allowlist secret -->" "${work}/ok.plg" ||
    fail "md5 entity not rewritten in place, or its pragma was lost"
grep -q "<!ENTITY sha256      \"${sha256}\"> <!-- pragma: allowlist secret -->" "${work}/ok.plg" ||
    fail "sha256 entity not rewritten in place, or its pragma was lost"
grep -q '<SHA256>&sha256;</SHA256>' "${work}/ok.plg" || fail "bundle FILE element lost its <SHA256>"
[ "$(diff "$PLG" "${work}/ok.plg" | grep -c '^>')" -eq 2 ] || fail "update changed lines other than the two checksums"
bash "$UPDATE" "$md5" "$sha256" "$version" "${work}/ok.plg" >/dev/null || fail "re-running with the same values failed"
echo "✓ Rewrites both checksums in place, keeps the pragmas, and is idempotent"

# Each case must fail and leave the file unchanged.
expect_failure() {
    local name="$1"
    shift
    cp "$PLG" "${work}/${name}.plg"
    if bash "$UPDATE" "$@" "${work}/${name}.plg" >/dev/null 2>&1; then
        fail "${name}: update succeeded but should have failed"
    fi
    cmp -s "$PLG" "${work}/${name}.plg" || fail "${name}: file changed despite the failure"
}
expect_failure bad-md5 "XYZ" "$sha256" "$version"
expect_failure bad-sha256 "$md5" "${sha256:0:63}" "$version"
expect_failure uppercase-sha256 "$md5" "$(printf '%064d' 3 | tr 0 B)" "$version"
expect_failure wrong-version "$md5" "$sha256" "1999.01.01"
echo "✓ Rejects malformed checksums and a version that does not match the PLG"

sed '/<!ENTITY sha256/d' "$PLG" >"${work}/missing.plg"
if bash "$UPDATE" "$md5" "$sha256" "$version" "${work}/missing.plg" >/dev/null 2>&1; then
    fail "missing-entity: update succeeded without a sha256 entity"
fi
echo "✓ Fails when the sha256 entity is missing"
