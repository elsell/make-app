#!/usr/bin/env bash
set -euo pipefail

readonly root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly subject="$root/scripts/check-hosted-android-toolchain.sh"
readonly test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

mkdir -p "$test_dir/bin" "$test_dir/sdk/platforms/android-36" \
  "$test_dir/sdk/build-tools/36.0.0" "$test_dir/sdk/cmake/3.22.1/bin" \
  "$test_dir/sdk/cmdline-tools/latest/bin"
for ndk in 27.1.12297006 27.0.12077973; do
  mkdir -p "$test_dir/sdk/ndk/$ndk/toolchains/llvm/prebuilt/linux-x86_64/bin"
  touch "$test_dir/sdk/ndk/$ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/clang"
done
touch "$test_dir/sdk/platforms/android-36/android.jar" \
  "$test_dir/sdk/build-tools/36.0.0/aapt2" \
  "$test_dir/sdk/cmake/3.22.1/bin/cmake" \
  "$test_dir/sdk/cmake/3.22.1/bin/ninja"
chmod +x "$test_dir/sdk/cmake/3.22.1/bin/ninja"

cat >"$test_dir/bin/java" <<'SH'
#!/usr/bin/env bash
echo 'openjdk version "17.0.15"' >&2
SH
cat >"$test_dir/sdk/cmdline-tools/latest/bin/sdkmanager" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$SDK_ARGUMENTS"
SH
# Keep the install test isolated even when the host has a real sdkmanager.
for tool in bash uname sed head; do
  ln -s "$(command -v "$tool")" "$test_dir/bin/$tool"
done
chmod +x "$test_dir/bin/java" "$test_dir/sdk/cmdline-tools/latest/bin/sdkmanager"

ANDROID_SDK_ROOT="$test_dir/sdk" PATH="$test_dir/bin" "$subject" >"$test_dir/valid.out"
grep -q 'hosted Android toolchain is complete' "$test_dir/valid.out"

SDK_ARGUMENTS="$test_dir/arguments" ANDROID_SDK_ROOT="$test_dir/sdk" PATH="$test_dir/bin" "$subject" --install >"$test_dir/install.out"
for package in 'platforms;android-36' 'build-tools;36.0.0' 'cmake;3.22.1' 'ndk;27.1.12297006' 'ndk;27.0.12077973'; do
  grep -Fxq "$package" "$test_dir/arguments"
done

rm "$test_dir/sdk/ndk/27.0.12077973/toolchains/llvm/prebuilt/linux-x86_64/bin/clang"
if ANDROID_SDK_ROOT="$test_dir/sdk" PATH="$test_dir/bin" "$subject" >"$test_dir/invalid.out" 2>&1; then
  echo "missing NDK input must fail closed" >&2
  exit 1
fi
grep -q 'missing .*27.0.12077973' "$test_dir/invalid.out"
