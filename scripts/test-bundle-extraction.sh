#!/bin/bash
# Regression tests for plugin bundle ownership (#242).
#
# Usage:
#   sudo bash scripts/test-bundle-extraction.sh
#       Runs the .plg extraction command against an archive owned by uid 1001
#       and checks that pre-existing directories keep their owner and mode and
#       that extracted files are owned by root. Requires root and GNU tar.
#   bash scripts/test-bundle-extraction.sh check-owners <bundle.tgz>
#       Fails if any archive entry is not owned by 0/0.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PLG="${REPO_ROOT}/unraid-management-agent.plg"
PLUGIN_REL="usr/local/emhttp/plugins/unraid-management-agent"

fail() {
    echo "✗ $*" >&2
    exit 1
}

check_owners() {
    bundle="$1"
    [ -f "$bundle" ] || fail "Bundle not found: $bundle"
    # GNU tar lists "uid/gid" in field 2; bsdtar lists uid and gid in fields 3 and 4
    bad=$(tar --numeric-owner -tvzf "$bundle" |
        awk '{ o = ($2 ~ /\//) ? $2 : $3 "/" $4; if (o != "0/0") print }')
    [ -z "$bad" ] || fail "Entries not owned by 0/0 in ${bundle}:
${bad}"
    echo "✓ All entries in ${bundle} are owned by 0/0"
}

assert_stat() {
    path="$1"
    want="$2"
    got=$(stat -c '%u:%g %a' "$path")
    [ "$got" = "$want" ] || fail "${path}: expected ${want}, got ${got}"
}

test_extraction() {
    [ "$(id -u)" -eq 0 ] || fail "Extraction test must run as root (use sudo)"

    work=$(mktemp -d)
    trap 'rm -rf "$work"' EXIT HUP INT TERM

    # Build an archive with the release layout, owned by the CI runner uid
    src="${work}/src"
    mkdir -p "${src}/${PLUGIN_REL}"
    echo "payload" >"${src}/${PLUGIN_REL}/VERSION"
    find "${src}/usr" -type d -exec chmod 0755 {} +
    archive="${work}/bundle.tgz"
    tar --owner=1001 --group=1001 --numeric-owner -czf "$archive" -C "$src" usr/

    # Pre-existing system directories with owners and modes the archive must not touch
    root="${work}/root"
    mkdir -p "${root}/${PLUGIN_REL}"
    chmod 0751 "${root}/usr" "${root}/usr/local" "${root}/usr/local/emhttp"
    chown 2002:2002 "${root}/usr/local/emhttp/plugins"
    chmod 0750 "${root}/usr/local/emhttp/plugins"

    # Run the production extraction line from the .plg, retargeted at the fixture root
    cmd=$(grep -E '^[[:space:]]*tar .*-xzf .*BUNDLE.* -C / ' "$PLG" || true)
    [ "$(printf '%s\n' "$cmd" | grep -c .)" -eq 1 ] || fail "Expected exactly one extraction line in ${PLG}"
    # shellcheck disable=SC2016 # literal match against the .plg line
    cmd=$(printf '%s' "$cmd" | sed 's| -C / | -C "${FIXTURE_ROOT}" |')
    echo "Running: ${cmd}"
    BUNDLE="$archive" FIXTURE_ROOT="$root" bash -c "$cmd" || fail "Extraction failed"

    assert_stat "${root}/usr" "0:0 751"
    assert_stat "${root}/usr/local" "0:0 751"
    assert_stat "${root}/usr/local/emhttp" "0:0 751"
    assert_stat "${root}/usr/local/emhttp/plugins" "2002:2002 750"
    assert_stat "${root}/${PLUGIN_REL}" "0:0 755"
    [ "$(cat "${root}/${PLUGIN_REL}/VERSION")" = "payload" ] || fail "Extracted file content mismatch"
    assert_stat "${root}/${PLUGIN_REL}/VERSION" "0:0 644"

    echo "✓ Extraction preserved existing directories and installed files as root"
}

case "${1:-}" in
check-owners)
    [ $# -eq 2 ] || fail "Usage: $0 check-owners <bundle.tgz>"
    check_owners "$2"
    ;;
"")
    test_extraction
    ;;
*)
    fail "Unknown mode: $1"
    ;;
esac
