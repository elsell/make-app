#!/usr/bin/env bash
set -euo pipefail

readonly source_sha="${1:?usage: $0 SOURCE_SHA TAG DIST_DIR}"
readonly tag="${2:?usage: $0 SOURCE_SHA TAG DIST_DIR}"
readonly dist_dir="${3:?usage: $0 SOURCE_SHA TAG DIST_DIR}"
readonly maximum_attempts="${RELEASE_UPLOAD_ATTEMPTS:-5}"
readonly retry_delay_seconds="${RELEASE_RETRY_DELAY_SECONDS:-5}"

if [[ ! "$source_sha" =~ ^[0-9a-f]{40}$ || ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "release publication requires an exact source SHA and semantic-version tag" >&2
  exit 2
fi
if [[ "$(git rev-parse HEAD)" != "$source_sha" ]]; then
  echo "release source is not the checked-out commit: $source_sha" >&2
  exit 1
fi

readonly assets=(
  "$dist_dir/make-app_linux_amd64.tar.gz"
  "$dist_dir/make-app_linux_arm64.tar.gz"
  "$dist_dir/make-app_darwin_amd64.tar.gz"
  "$dist_dir/make-app_darwin_arm64.tar.gz"
  "$dist_dir/checksums.txt"
)
for asset in "${assets[@]}"; do
  if [[ ! -s "$asset" ]]; then
    echo "release asset is missing or empty: $asset" >&2
    exit 1
  fi
done

retry() {
  local attempt=1
  while ! "$@"; do
    if ((attempt >= maximum_attempts)); then
      echo "release command failed after $maximum_attempts attempts: $*" >&2
      return 1
    fi
    echo "release command failed; retrying attempt $((attempt + 1))/$maximum_attempts: $*" >&2
    sleep "$retry_delay_seconds"
    attempt=$((attempt + 1))
  done
}

if local_tag_sha="$(git rev-parse "${tag}^{commit}" 2>/dev/null)"; then
  if [[ "$local_tag_sha" != "$source_sha" ]]; then
    echo "existing local tag $tag resolves to $local_tag_sha, not $source_sha" >&2
    exit 1
  fi
else
  git tag "$tag" "$source_sha"
fi

ensure_remote_tag() {
  local remote_tag_sha
  remote_tag_sha="$(git ls-remote --tags origin "refs/tags/$tag" | awk 'NR == 1 { print $1 }')"
  if [[ -n "$remote_tag_sha" ]]; then
    [[ "$remote_tag_sha" = "$source_sha" ]] || {
      echo "existing remote tag $tag resolves to $remote_tag_sha, not $source_sha" >&2
      return 1
    }
    return 0
  fi
  git push origin "$tag"
}
retry ensure_remote_tag

ensure_draft_release() {
  local draft
  if draft="$(gh release view "$tag" --json isDraft --jq .isDraft 2>/dev/null)"; then
    if [[ "$draft" != "true" ]]; then
      gh release edit "$tag" --draft=true
    fi
    return 0
  fi
  gh release create "$tag" --target "$source_sha" --generate-notes --draft
}
retry ensure_draft_release

for asset in "${assets[@]}"; do
  retry gh release upload "$tag" "$asset" --clobber
done
retry gh release edit "$tag" --draft=false

echo "Published $tag from $source_sha with ${#assets[@]} verified assets"
