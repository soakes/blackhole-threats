#!/usr/bin/env bash
set -euo pipefail

previous_tag="${1:-}"
to_ref="${2:-HEAD}"
output_path="${3:-/dev/stdout}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

sections=(
  breaking
  features
  fixes
  packaging
  docs
  ci
  deps
  maintenance
)

conventional_regex='^([[:alnum:]][[:alnum:]-]*)(\([^)]+\))?(!)?:[[:space:]]+(.*)$'

for section in "${sections[@]}"; do
  : > "${tmpdir}/${section}.md"
done

if [ -n "${previous_tag}" ]; then
  log_range="${previous_tag}..${to_ref}"
else
  log_range="${to_ref}"
fi

commit_count="$(git rev-list --count "${log_range}")"

append_commit() {
  local file="$1"
  local subject="$2"
  local short_hash="$3"
  local body="$4"

  printf -- '- %s (`%s`)\n' "${subject}" "${short_hash}" >> "${file}"

  if [ -n "${body}" ]; then
    while IFS= read -r line; do
      if [ -n "${line}" ]; then
        printf '  %s\n' "${line}" >> "${file}"
      fi
    done <<<"${body}"
  fi

  printf '\n' >> "${file}"
}

while IFS= read -r -d $'\036' record; do
  [ -n "${record}" ] || continue
  record="${record#$'\n'}"

  hash="${record%%$'\037'*}"
  rest="${record#*$'\037'}"
  subject="${rest%%$'\037'*}"
  body="${rest#*$'\037'}"

  short_hash="${hash:0:7}"
  section="maintenance"
  clean_subject="${subject}"
  breaking=false

  if [[ "${subject}" =~ ${conventional_regex} ]]; then
    commit_type="${BASH_REMATCH[1]}"
    bang="${BASH_REMATCH[3]}"
    clean_subject="${BASH_REMATCH[4]}"

    case "${commit_type}" in
      feat)
        section="features"
        ;;
      fix|perf)
        section="fixes"
        ;;
      container|build|packaging)
        section="packaging"
        ;;
      docs)
        section="docs"
        ;;
      ci)
        section="ci"
        ;;
      deps)
        section="deps"
        ;;
      *)
        section="maintenance"
        ;;
    esac

    if [ -n "${bang}" ]; then
      breaking=true
    fi
  fi

  if grep -q 'BREAKING CHANGE:' <<<"${body}"; then
    breaking=true
  fi

  body="$(printf '%s\n' "${body}" | sed '/^[[:space:]]*$/d')"

  append_commit "${tmpdir}/${section}.md" "${clean_subject}" "${short_hash}" "${body}"

  if [ "${breaking}" = true ]; then
    append_commit "${tmpdir}/breaking.md" "${clean_subject}" "${short_hash}" "${body}"
  fi
done < <(git log --reverse --format='%H%x1f%s%x1f%b%x1e' "${log_range}")

{
  printf '# Release %s\n\n' "$(git describe --tags --exact-match "${to_ref}" 2>/dev/null || printf '%s' "${to_ref}")"

  if [ -n "${previous_tag}" ]; then
    printf 'Changes since `%s`.\n\n' "${previous_tag}"
  else
    printf 'Initial release.\n\n'
  fi

  printf 'Included commits: %s\n\n' "${commit_count}"

  if [ -s "${tmpdir}/breaking.md" ]; then
    printf '## Breaking Changes\n\n'
    cat "${tmpdir}/breaking.md"
  fi
  if [ -s "${tmpdir}/features.md" ]; then
    printf '## Features\n\n'
    cat "${tmpdir}/features.md"
  fi
  if [ -s "${tmpdir}/fixes.md" ]; then
    printf '## Fixes\n\n'
    cat "${tmpdir}/fixes.md"
  fi
  if [ -s "${tmpdir}/packaging.md" ]; then
    printf '## Packaging and Delivery\n\n'
    cat "${tmpdir}/packaging.md"
  fi
  if [ -s "${tmpdir}/docs.md" ]; then
    printf '## Documentation\n\n'
    cat "${tmpdir}/docs.md"
  fi
  if [ -s "${tmpdir}/ci.md" ]; then
    printf '## CI and Automation\n\n'
    cat "${tmpdir}/ci.md"
  fi
  if [ -s "${tmpdir}/deps.md" ]; then
    printf '## Dependencies\n\n'
    cat "${tmpdir}/deps.md"
  fi
  if [ -s "${tmpdir}/maintenance.md" ]; then
    printf '## Maintenance\n\n'
    cat "${tmpdir}/maintenance.md"
  fi
} > "${output_path}"
