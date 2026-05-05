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
  ci
  docs
  deps
  maintenance
)

conventional_regex='^([[:alnum:]][[:alnum:]-]*)(\([^)]+\))?(!)?:[[:space:]]+(.*)$'

for section in "${sections[@]}"; do
  : > "${tmpdir}/${section}.md"
done

repo_url=''

resolve_repo_url() {
  local remote_url=''

  if [ -n "${GITHUB_SERVER_URL:-}" ] && [ -n "${GITHUB_REPOSITORY:-}" ]; then
    printf '%s/%s\n' "${GITHUB_SERVER_URL%/}" "${GITHUB_REPOSITORY}"
    return
  fi

  remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"

  case "${remote_url}" in
    https://github.com/*|http://github.com/*)
      printf '%s\n' "${remote_url%.git}"
      ;;
    git@github.com:*)
      printf 'https://github.com/%s\n' "${remote_url#git@github.com:}" | sed 's/\.git$//'
      ;;
    ssh://git@github.com/*)
      printf 'https://github.com/%s\n' "${remote_url#ssh://git@github.com/}" | sed 's/\.git$//'
      ;;
  esac
}

repo_url="$(resolve_repo_url)"

if [ -n "${previous_tag}" ]; then
  log_range="${previous_tag}..${to_ref}"
else
  log_range="${to_ref}"
fi

tag_name="$(git describe --tags --exact-match "${to_ref}" 2>/dev/null || printf '%s' "${to_ref}")"

format_commit_ref() {
  local hash="$1"
  local short_hash="$2"

  if [ -n "${repo_url}" ]; then
    printf '[`%s`](%s/commit/%s)\n' "${short_hash}" "${repo_url}" "${hash}"
  else
    printf '`%s`\n' "${short_hash}"
  fi
}

format_tag_ref() {
  local tag="$1"

  if [ -n "${repo_url}" ]; then
    printf '[`%s`](%s/releases/tag/%s)\n' "${tag}" "${repo_url}" "${tag}"
  else
    printf '`%s`\n' "${tag}"
  fi
}

format_compare_ref() {
  local from_ref="$1"
  local to_ref_name="$2"

  if [ -n "${repo_url}" ]; then
    printf '[`%s...%s`](%s/compare/%s...%s)\n' \
      "${from_ref}" \
      "${to_ref_name}" \
      "${repo_url}" \
      "${from_ref}" \
      "${to_ref_name}"
  else
    printf '`%s...%s`\n' "${from_ref}" "${to_ref_name}"
  fi
}

trim() {
  sed 's/^[[:space:]]*//; s/[[:space:]]*$//'
}

normalise_space() {
  tr '\n' ' ' | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//'
}

extract_merge_subject() {
  local body="$1"
  local line=''
  local first_seen=false

  while IFS= read -r line; do
    line="$(printf '%s' "${line}" | trim)"
    if [ -z "${line}" ]; then
      continue
    fi

    if [ "${first_seen}" = false ]; then
      first_seen=true
      continue
    fi

    printf '%s\n' "${line}"
    return
  done <<<"${body}"
}

format_subject() {
  local subject="$1"

  if [ -z "${subject}" ]; then
    return
  fi

  printf '%s%s\n' \
    "$(printf '%s' "${subject:0:1}" | tr '[:lower:]' '[:upper:]')" \
    "${subject:1}"
}

clean_subject() {
  local subject="$1"
  local body="$2"
  local clean="${subject}"
  local merge_subject=''

  if [[ "${clean}" =~ ^Merge\ pull\ request\ #[0-9]+ ]]; then
    merge_subject="$(extract_merge_subject "${body}")"
    if [ -n "${merge_subject}" ]; then
      clean="${merge_subject}"
    fi
  fi

  if [[ "${clean}" =~ ${conventional_regex} ]]; then
    clean="${BASH_REMATCH[4]}"
  fi

  printf '%s\n' "$(format_subject "${clean}")"
}

filter_body_lines() {
  local body="$1"
  local line=''

  while IFS= read -r line; do
    line="$(printf '%s' "${line}" | trim)"

    if [ -z "${line}" ]; then
      printf '\n'
      continue
    fi

    case "${line}" in
      Merge\ pull\ request\ #*|Signed-off-by:*|Co-authored-by:*|Reviewed-by:*|Acked-by:*|Tested-by:*|Refs:*|Ref:*|Fixes:*|Closes:*|Resolves:*)
        continue
        ;;
      Release\ notes|Commits|updated-dependencies:|dependency-name:*|dependency-version:*|dependency-type:*|update-type:*|...\ )
        continue
        ;;
    esac

    if [[ "${line}" =~ ^https?:// ]]; then
      continue
    fi

    printf '%s\n' "${line}"
  done <<<"${body}"
}

extract_summary() {
  local body="$1"
  local cleaned=''
  local paragraph=''
  local line=''

  cleaned="$(filter_body_lines "${body}")"

  while IFS= read -r line; do
    if [ -z "${line}" ]; then
      if [ -n "${paragraph}" ]; then
        break
      fi
      continue
    fi

    case "${line}" in
      -\ *|\*\ *)
        if [ -n "${paragraph}" ]; then
          break
        fi
        paragraph="${line#??}"
        break
        ;;
      *)
        if [ -n "${paragraph}" ]; then
          paragraph="${paragraph} ${line}"
        else
          paragraph="${line}"
        fi
        ;;
    esac
  done <<<"${cleaned}"

  paragraph="$(printf '%s' "${paragraph}" | normalise_space)"

  case "${paragraph}" in
    ""|Bumps\ *|Version\ bump\ only*)
      return
      ;;
  esac

  case "${paragraph}" in
    *". "*) paragraph="${paragraph%%. *}." ;;
    *"? "*) paragraph="${paragraph%%\? *}?" ;;
    *"! "*) paragraph="${paragraph%%\! *}!" ;;
  esac

  if [ "${#paragraph}" -gt 220 ]; then
    paragraph="${paragraph:0:217}..."
  fi

  printf '%s\n' "${paragraph}"
}

append_commit() {
  local file="$1"
  local hash="$2"
  local subject="$3"
  local body="$4"
  local summary=''
  local ref=''

  ref="$(format_commit_ref "${hash}" "${hash:0:7}")"
  summary="$(extract_summary "${body}")"

  printf -- '- %s (%s)\n' "${subject}" "${ref}" >> "${file}"

  if [ -n "${summary}" ]; then
    printf '  - %s\n' "${summary}" >> "${file}"
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

  if [[ "${subject}" =~ ^Merge\ pull\ request\ #[0-9]+ ]]; then
    continue
  fi

  if [[ "${subject}" =~ ^chore\(release\):[[:space:]]+update\ Debian\ changelog\ for\ v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    continue
  fi

  section="maintenance"
  breaking=false
  cleaned_subject="$(clean_subject "${subject}" "${body}")"

  if [[ "${subject}" =~ ${conventional_regex} ]]; then
    commit_type="${BASH_REMATCH[1]}"
    bang="${BASH_REMATCH[3]}"

    case "${commit_type}" in
      feat)
        section="features"
        ;;
      fix|perf|revert)
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
    section="breaking"
  fi

  append_commit "${tmpdir}/${section}.md" "${hash}" "${cleaned_subject}" "${body}"

  if [ "${breaking}" = true ] && [ "${section}" != "breaking" ]; then
    append_commit "${tmpdir}/breaking.md" "${hash}" "${cleaned_subject}" "${body}"
  fi
done < <(git log --reverse --format='%H%x1f%s%x1f%b%x1e' "${log_range}")

compare_ref=''
previous_tag_ref=''
release_tag_ref=''
release_target="${tag_name}"
release_label='Release'

if [ -n "${previous_tag}" ]; then
  compare_ref="$(format_compare_ref "${previous_tag}" "${tag_name}")"
  previous_tag_ref="$(format_tag_ref "${previous_tag}")"
fi

release_tag_ref="$(format_tag_ref "${tag_name}")"

if [[ "${tag_name}" =~ ^(v[0-9]+\.[0-9]+\.[0-9]+)-rc\.[0-9]+$ ]]; then
  release_target="${BASH_REMATCH[1]}"
  release_label='Release candidate'
fi

{
  printf '## Release Overview\n\n'
  printf '%s for `%s`.\n\n' "${release_label}" "${release_target}"
  printf -- '- Published tag: %s\n' "${release_tag_ref}"

  if [ -n "${compare_ref}" ]; then
    printf -- '- Full diff: %s\n' "${compare_ref}"
  fi
  printf '\n'

  if [ -s "${tmpdir}/breaking.md" ]; then
    printf '## Breaking Changes\n\n'
    cat "${tmpdir}/breaking.md"
  fi

  printf '## Included Changes\n\n'

  if [ -s "${tmpdir}/features.md" ]; then
    printf '### Features\n\n'
    cat "${tmpdir}/features.md"
  fi
  if [ -s "${tmpdir}/fixes.md" ]; then
    printf '### Fixes\n\n'
    cat "${tmpdir}/fixes.md"
  fi
  if [ -s "${tmpdir}/packaging.md" ]; then
    printf '### Packaging and Delivery\n\n'
    cat "${tmpdir}/packaging.md"
  fi
  if [ -s "${tmpdir}/ci.md" ]; then
    printf '### CI and Automation\n\n'
    cat "${tmpdir}/ci.md"
  fi
  if [ -s "${tmpdir}/deps.md" ]; then
    printf '### Dependencies\n\n'
    cat "${tmpdir}/deps.md"
  fi
  if [ -s "${tmpdir}/docs.md" ]; then
    printf '### Documentation\n\n'
    cat "${tmpdir}/docs.md"
  fi
  if [ -s "${tmpdir}/maintenance.md" ]; then
    printf '### Maintenance\n\n'
    cat "${tmpdir}/maintenance.md"
  fi

  printf '## Published Artifacts\n\n'
  printf -- '- Release binaries for `linux/amd64`, `linux/arm64`, and `linux/arm`\n'
  printf -- '- Debian packages for `amd64`, `arm64`, and `armhf`, plus a Debian source package\n'
  printf -- '- Checksums attached to the release\n'
  printf -- '- Container images published to `ghcr.io/soakes/blackhole-threats`\n'
  printf -- '- Stable releases also publish the signed GitHub Pages APT repository\n'
} > "${output_path}"
