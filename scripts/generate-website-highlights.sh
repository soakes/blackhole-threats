#!/usr/bin/env bash
set -euo pipefail

to_ref="${1:-HEAD}"
count="${2:-3}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

if [[ ! "${count}" =~ ^[0-9]+$ ]]; then
  echo "highlight count must be an integer, got: ${count}" >&2
  exit 1
fi

fallback_highlights() {
  cat <<'EOF'
Fetch feeds concurrently & normalize IPs
Summarise networks to communities
Export concise BGP route deltas
EOF
}

resolve_current_tag() {
  if [[ "${to_ref}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    printf '%s\n' "${to_ref}"
    return
  fi

  git tag --points-at "${to_ref}" --list 'v*' \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | head -n1 || true
}

extract_highlights_from_markdown() {
  local input_path="$1"

  awk -v max_count="${count}" '
    function trim(value) {
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      return value
    }

    function emit(value) {
      value = trim(value)
      if (value == "" || seen[value]++) {
        return
      }

      print value
      emitted++
      if (emitted >= max_count) {
        exit
      }
    }

    /^## Included Changes/ {
      in_changes = 1
      next
    }

    /^## Published Artifacts/ {
      exit
    }

    !in_changes {
      next
    }

    /^### / {
      next
    }

    /^  - / {
      next
    }

    /^- / {
      line = substr($0, 3)
      gsub(/\(\[[^]]+\]\([^)]+\)\)$/, "", line)
      gsub(/\[[^]]+\]\([^)]+\)/, "", line)
      gsub(/`/, "", line)
      gsub(/[[:space:]]+$/, "", line)
      emit(line)
    }
  ' "${input_path}"
}

current_tag="$(resolve_current_tag)"

previous_tag="$(
  git for-each-ref --merged "${to_ref}" --sort=-v:refname --format='%(refname:short)' refs/tags/v* \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | grep -vx "${current_tag}" \
    | head -n1 || true
)"

release_notes_path="${tmpdir}/release-notes.md"

if [ -n "${current_tag}" ] && command -v gh >/dev/null 2>&1; then
  if gh release view "${current_tag}" --json body --jq .body > "${release_notes_path}" 2>/dev/null; then
    :
  fi
fi

if [ ! -s "${release_notes_path}" ]; then
  bash scripts/generate-release-notes.sh "${previous_tag}" "${to_ref}" "${release_notes_path}"
fi

highlights="$(extract_highlights_from_markdown "${release_notes_path}" || true)"

if [ -n "${highlights}" ]; then
  printf '%s\n' "${highlights}"
  exit 0
fi

fallback_highlights
