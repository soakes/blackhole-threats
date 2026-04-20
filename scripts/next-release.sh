#!/usr/bin/env bash
set -euo pipefail

latest_tag="$(
  git for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags/v* \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | head -n1 || true
)"
initial_version="$(
  sed -n '1s/^[^(]*(\([^)]*\)).*/\1/p' debian/changelog 2>/dev/null | sed 's/-[^-]*$//' || true
)"

if [ -n "${latest_tag}" ]; then
  commit_count="$(git rev-list --count "${latest_tag}..HEAD")"
else
  commit_count="$(git rev-list --count HEAD)"
fi

if [ "${commit_count}" -eq 0 ]; then
  cat <<EOF
release_required=false
previous_tag=${latest_tag}
next_tag=${latest_tag}
bump=none
commit_count=0
EOF
  exit 0
fi

bump_level="$(
  if [ -n "${latest_tag}" ]; then
    git log --format='%s%n%b%x1e' "${latest_tag}..HEAD"
  else
    git log --format='%s%n%b%x1e' HEAD
  fi | awk '
    BEGIN { RS = "\036"; level = 0 }
    {
      line_count = split($0, lines, /\n/)
      subject = ""
      for (i = 1; i <= line_count; i++) {
        if (lines[i] != "") {
          subject = lines[i]
          break
        }
      }

      if ($0 ~ /BREAKING CHANGE:/ || subject ~ /^[[:alnum:]][[:alnum:]-]*(\([^)]+\))?!: /) {
        if (level < 3) {
          level = 3
        }
        next
      }

      if (subject ~ /^feat(\([^)]+\))?: /) {
        if (level < 2) {
          level = 2
        }
        next
      }

      if (subject ~ /^(fix|perf|revert|container|build|deps|packaging)(\([^)]+\))?: /) {
        if (level < 1) {
          level = 1
        }
      }
    }
    END { print level }
  '
)"

if [ -z "${latest_tag}" ]; then
  if [[ "${initial_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    next_tag="v${initial_version}"
    bump="bootstrap"
  else
    case "${bump_level}" in
      3|2)
        next_tag="v0.1.0"
        bump="minor"
        ;;
      1)
        next_tag="v0.0.1"
        bump="patch"
        ;;
      *)
        cat <<EOF
release_required=false
previous_tag=
next_tag=
bump=none
commit_count=${commit_count}
EOF
        exit 0
        ;;
    esac
  fi
else
  if [[ ! "${latest_tag}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    echo "latest tag does not use v<major>.<minor>.<patch>: ${latest_tag}" >&2
    exit 1
  fi

  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"

  case "${bump_level}" in
    3)
      major="$((major + 1))"
      minor=0
      patch=0
      bump="major"
      ;;
    2)
      minor="$((minor + 1))"
      patch=0
      bump="minor"
      ;;
    1)
      patch="$((patch + 1))"
      bump="patch"
      ;;
    *)
      cat <<EOF
release_required=false
previous_tag=${latest_tag}
next_tag=
bump=none
commit_count=${commit_count}
EOF
      exit 0
      ;;
  esac

  next_tag="v${major}.${minor}.${patch}"
fi

cat <<EOF
release_required=true
previous_tag=${latest_tag}
next_tag=${next_tag}
bump=${bump}
commit_count=${commit_count}
EOF
