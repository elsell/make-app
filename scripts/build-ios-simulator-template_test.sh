#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/apps/mobile/ios/HabitKit.xcworkspace" "$fixture/bin"
cp "$root/template/base/scripts/build-ios-simulator.sh" "$fixture/build-ios-simulator.sh"
chmod +x "$fixture/build-ios-simulator.sh"
cat >"$fixture/bin/xcodebuild" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ " $* " == *" -list "* ]]; then
  printf '%s\n' "${FAKE_SCHEMES_JSON:?}"
  exit 0
fi
printf '%s\n' "$*" >"${FAKE_XCODEBUILD_LOG:?}"
EOF
chmod +x "$fixture/bin/xcodebuild"

export PATH="$fixture/bin:$PATH"
export FAKE_XCODEBUILD_LOG="$fixture/xcodebuild.log"
export FAKE_SCHEMES_JSON='{"workspace":{"schemes":["EASClient","HabitKit","Pods-HabitKit"]}}'

(
  cd "$fixture"
  ./build-ios-simulator.sh
)

grep -Fq -- '-scheme HabitKit ' "$FAKE_XCODEBUILD_LOG"
if grep -Fq -- '-scheme EASClient ' "$FAKE_XCODEBUILD_LOG"; then
  echo "dependency scheme was selected instead of the application scheme" >&2
  exit 1
fi

rm -f "$FAKE_XCODEBUILD_LOG"
export FAKE_SCHEMES_JSON='{"workspace":{"schemes":["EASClient","Pods-HabitKit"]}}'
if (
  cd "$fixture"
  ./build-ios-simulator.sh
) >"$fixture/missing.out" 2>&1; then
  echo "missing application scheme passed" >&2
  exit 1
fi
grep -Fq 'iOS application scheme "HabitKit" missing' "$fixture/missing.out"
test ! -e "$FAKE_XCODEBUILD_LOG"
