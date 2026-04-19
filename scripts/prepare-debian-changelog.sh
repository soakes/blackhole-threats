#!/usr/bin/env bash
set -euo pipefail

release_version="${1:?release version is required}"
suite="${2:-stable}"

release_version="${release_version#v}"
package_name="$(dpkg-parsechangelog -SSource)"
urgency="$(dpkg-parsechangelog -SUrgency)"
maintainer="$(dpkg-parsechangelog -SMaintainer)"
timestamp="$(date -R)"
tmpfile="$(mktemp)"

awk \
  -v header="${package_name} (${release_version}-1) ${suite}; urgency=${urgency}" \
  -v trailer=" -- ${maintainer}  ${timestamp}" \
  '
  NR == 1 {
    print header
    next
  }

  !updated && /^ -- / {
    print trailer
    updated = 1
    next
  }

  {
    print
  }
  ' debian/changelog > "${tmpfile}"

mv "${tmpfile}" debian/changelog
