#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.9.0}"
API="${2:-x5.0}"
MODULE="github.com/flippyboy/packer-plugin-kubevirt/version"
LDFLAGS="-s -w -X ${MODULE}.Version=${VERSION}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="${ROOT}/dist"
rm -rf "${DIST}"
mkdir -p "${DIST}"

zip_files() {
  local archive="$1"
  shift
  python3 - "$archive" "$@" <<'PY'
import sys, zipfile
archive, *files = sys.argv[1:]
with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for path in files:
        zf.write(path, arcname=path.split("/")[-1])
PY
}

platforms=(
  "darwin:amd64"
  "darwin:arm64"
  "linux:386"
  "linux:amd64"
  "linux:arm"
  "linux:arm64"
  "freebsd:386"
  "freebsd:amd64"
  "freebsd:arm"
  "netbsd:386"
  "netbsd:amd64"
  "netbsd:arm"
  "openbsd:386"
  "openbsd:amd64"
  "openbsd:arm"
  "solaris:amd64"
  "windows:386"
  "windows:amd64"
)

for platform in "${platforms[@]}"; do
  GOOS="${platform%%:*}"
  GOARCH="${platform##*:}"
  BIN="packer-plugin-kubevirt_v${VERSION}_${API}_${GOOS}_${GOARCH}"
  if [[ "${GOOS}" == "windows" ]]; then
    BIN="${BIN}.exe"
  fi

  echo "Building ${GOOS}/${GOARCH}..."
  env CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
    go build -trimpath -buildvcs=false -ldflags="${LDFLAGS}" -o "${DIST}/${BIN}" "${ROOT}"

  cp "${ROOT}/LICENSE" "${DIST}/LICENSE.txt"
  ZIP="${DIST}/packer-plugin-kubevirt_${VERSION}_${GOOS}_${GOARCH}.zip"
  zip_files "${ZIP}" "${DIST}/${BIN}" "${DIST}/LICENSE.txt"
  rm -f "${DIST}/${BIN}" "${DIST}/LICENSE.txt"
done

echo "Built $(ls "${DIST}"/*.zip | wc -l) release archives in ${DIST}"