#!/usr/bin/env bash
set -euo pipefail

archive="grype_0.111.1_linux_amd64.tar.gz"
expected_sha256="2bc0bc60f1f4e10b0429f5e84517ec4cf0d769d2ef66875c64fc6640e136fd8f"
download_url="https://github.com/anchore/grype/releases/download/v0.111.1/grype_0.111.1_linux_amd64.tar.gz"

if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  echo "Grype bootstrap supports only the reviewed Linux amd64 release runner" >&2
  exit 1
fi
for command in curl sha256sum tar install; do
  command -v "$command" >/dev/null || {
    echo "Grype bootstrap requires $command" >&2
    exit 1
  }
done

temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT
archive_path="$temporary_directory/$archive"
extract_path="$temporary_directory/extract"
mkdir -p "$extract_path" .bin

curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --silent --show-error \
  --output "$archive_path" "$download_url"
printf '%s  %s\n' "$expected_sha256" "$archive_path" | sha256sum --check --status
tar --extract --gzip --file "$archive_path" --directory "$extract_path"
install --mode 0755 "$extract_path/grype" ./.bin/grype

version_output="$(./.bin/grype version)"
grep -Eq '^Version:[[:space:]]+0\.111\.1$' <<<"$version_output" || {
  echo "Installed Grype did not report the reviewed version 0.111.1" >&2
  exit 1
}
