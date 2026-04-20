#!/usr/bin/env bash
set -euo pipefail

to_ref="${1:-HEAD}"
count="${2:-3}"

if [[ ! "${count}" =~ ^[0-9]+$ ]]; then
  echo "highlight count must be an integer, got: ${count}" >&2
  exit 1
fi

current_tag="$(
  git tag --points-at "${to_ref}" --list 'v*' \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | head -n1 || true
)"

previous_tag="$(
  git for-each-ref --merged "${to_ref}" --sort=-v:refname --format='%(refname:short)' refs/tags/v* \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | grep -vx "${current_tag}" \
    | head -n1 || true
)"

if [ -n "${previous_tag}" ]; then
  log_range="${previous_tag}..${to_ref}"
else
  log_range="${to_ref}"
fi

highlights="$(
  git log --no-merges --format='%s' "${log_range}" \
    | awk '
      {
        subject = $0
        if (subject ~ /^Merge pull request /) {
          next
        }

        sub(/^([[:alnum:]][[:alnum:]-]*)(\([^)]+\))?(!)?:[[:space:]]*/, "", subject)
        gsub(/^[[:space:]]+|[[:space:]]+$/, "", subject)

        if (subject == "") {
          next
        }

        subject = toupper(substr(subject, 1, 1)) substr(subject, 2)
        print subject
      }
    ' \
    | awk '!seen[$0]++' \
    | head -n "${count}" || true
)"

if [ -n "${highlights}" ]; then
  printf '%s\n' "${highlights}"
  exit 0
fi

cat <<'EOF'
Fetch feeds concurrently & normalize IPs
Summarise networks to communities
Export concise BGP route deltas
EOF
