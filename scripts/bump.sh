#!/usr/bin/env bash
set -euo pipefail

remote="${BUMP_REMOTE:-origin}"
branch="${BUMP_BRANCH:-$(git branch --show-current)}"
bump_kind="${1:-patch}"
version_file="cmd/cfl/version.go"

perl -0777 -i -pe 's/(const\s+Version\s*=\s*")v([0-9]+\.[0-9]+\.[0-9]+)"/${1}${2}"/g' "$version_file"
gobump "$bump_kind" -w ./cmd/cfl

next_semver="$(sed -nE 's/^[[:space:]]*const[[:space:]]+Version[[:space:]]*=[[:space:]]*"([0-9]+\.[0-9]+\.[0-9]+)".*$/\1/p' "$version_file" | head -n 1)"

new_version="v${next_semver}"
perl -0777 -i -pe 's/(const\s+Version\s*=\s*")(?:v)?[0-9]+\.[0-9]+\.[0-9]+(")/${1}'"$new_version"'${2}/g' "$version_file"

tag="${new_version}"

git add "$version_file"
git commit -m "chore(bump): ${tag}"
git tag -a "${tag}" -m "${tag}"

git push "$remote" "$branch"
git push "$remote" "${tag}"

echo "Bumped to ${tag} and pushed ${branch} + tag to ${remote}."
