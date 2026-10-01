#!/usr/bin/env bash
set -euo pipefail

readonly sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
readonly required_ndks=("27.1.12297006" "27.0.12077973")

fail() {
  echo "hosted Android toolchain is incomplete: $*" >&2
  exit 1
}

[[ "$(uname -s)" == "Linux" && "$(uname -m)" == "x86_64" ]] ||
  fail "requires Linux x86_64"
[[ -n "$sdk_root" && -d "$sdk_root" ]] ||
  fail "ANDROID_SDK_ROOT or ANDROID_HOME must name an installed SDK"

sdkmanager_path="$(command -v sdkmanager || true)"
if [[ -z "$sdkmanager_path" ]]; then
  for candidate in \
    "$sdk_root/cmdline-tools/latest/bin/sdkmanager" \
    "$sdk_root/cmdline-tools/bin/sdkmanager" \
    "$sdk_root/tools/bin/sdkmanager"
  do
    if [[ -x "$candidate" ]]; then
      sdkmanager_path="$candidate"
      break
    fi
  done
fi
[[ -n "$sdkmanager_path" ]] || fail "sdkmanager is not available on PATH or in the installed SDK"
command -v java >/dev/null 2>&1 || fail "Java is not available on PATH"
java_major="$(java -version 2>&1 | sed -nE 's/.*version "([0-9]+)(\..*)?".*/\1/p' | head -n1)"
[[ "$java_major" == "17" ]] || fail "Java 17 is required (found ${java_major:-unknown})"

required_paths=(
  "$sdk_root/platforms/android-36/android.jar"
  "$sdk_root/build-tools/36.0.0/aapt2"
  "$sdk_root/cmake/3.22.1/bin/cmake"
)
for ndk in "${required_ndks[@]}"; do
  required_paths+=("$sdk_root/ndk/$ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/clang")
done
for path in "${required_paths[@]}"; do
  [[ -e "$path" ]] || fail "missing $path"
done

if ! command -v ninja >/dev/null 2>&1 && [[ ! -x "$sdk_root/cmake/3.22.1/bin/ninja" ]]; then
  fail "CMake 3.22.1 Ninja is not available"
fi

echo "hosted Android toolchain is complete: platform android-36, build tools 36.0.0, NDK ${required_ndks[0]} and ${required_ndks[1]}, CMake 3.22.1"
