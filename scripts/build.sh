#!/bin/bash
# build.sh is a local build helper that compiles subrelay for a
# single target (GOOS/GOARCH) with the correct CGO cross-compiler
# settings and optional Linux packaging.
#
# Usage:
#   build.sh <goos>/<goarch> [version] [--package]
#
# Examples:
#   build.sh linux/amd64 1.0.0 --package
#   build.sh windows/amd64 1.0.0
#   build.sh linux/arm64 1.0.0 --package
#
# The script expects the appropriate C cross-compiler to be
# installed and available on PATH. See the README for toolchain
# installation instructions per target.
#
# Args:
#   - target: GOOS/GOARCH pair (e.g. linux/amd64).
#   - version: semantic version string (defaults to "dev").
#   - --package: when set, produce .deb/.rpm/.tar.gz for Linux
#     targets using packaging/package-linux.sh.
#
# Errors:
#   - Exits with a non-zero status on invalid target, missing
#     compiler, or build failure.
set -euo pipefail

target="${1:?usage: build.sh <goos>/<goarch> [version] [--package]}"
version="dev"
do_package=false

for arg in "${@:2}"; do
    case "$arg" in
        --package) do_package=true ;;
        *) version="$arg" ;;
    esac
done

goos="${target%%/*}"
goarch="${target##*/}"

# Build tags required by the sing-box extended fork.
build_tags="with_utls,with_grpc,with_quic"

# Per-target compiler and flags configuration.
case "${goos}/${goarch}" in
    linux/amd64)
        cc="gcc"
        deb_arch="amd64"
        ;;
    linux/arm64)
        cc="aarch64-linux-gnu-gcc"
        deb_arch="arm64"
        export PKG_CONFIG_PATH="/usr/lib/aarch64-linux-gnu/pkgconfig"
        ;;
    linux/arm)
        cc="arm-linux-gnueabihf-gcc"
        deb_arch="armhf"
        goarm="7"
        export PKG_CONFIG_PATH="/usr/lib/arm-linux-gnueabihf/pkgconfig"
        ;;
    linux/386)
        cc="gcc"
        deb_arch="i386"
        export CGO_CFLAGS="-m32"
        export CGO_LDFLAGS="-m32"
        export PKG_CONFIG_PATH="/usr/lib/i386-linux-gnu/pkgconfig"
        ;;
    windows/amd64)
        cc="x86_64-w64-mingw32-gcc"
        mingw_prefix="x86_64-w64-mingw32"
        ;;
    windows/386)
        cc="i686-w64-mingw32-gcc"
        mingw_prefix="i686-w64-mingw32"
        ;;
    windows/arm64)
        cc="zig cc -target aarch64-windows-gnu"
        cxx="zig c++ -target aarch64-windows-gnu"
        use_zig=true
        mingw_prefix="aarch64-w64-mingw32"
        ;;
    *)
        echo "error: unsupported target: ${target}" >&2
        echo "supported: linux/amd64 linux/arm64 linux/arm linux/386" >&2
        echo "           windows/amd64 windows/386 windows/arm64" >&2
        exit 1
        ;;
esac

script_dir="$(cd "$(dirname "$0")" && pwd)"
project_root="$(cd "$script_dir/.." && pwd)"
cd "$project_root"

mkdir -p build

# Set common build environment.
export CGO_ENABLED=1
export GOOS="$goos"
export GOARCH="$goarch"
export CC="$cc"
export CXX="${cxx:-${cc/gcc/g++}}"
[[ -n "${goarm:-}" ]] && export GOARM="$goarm"

# Assemble ldflags: strip debug symbols, inject version, and hide
# the console window on Windows.
ldflags="-s -w -X subrelay/internal/update.CurrentVersion=${version}"
if [[ "$goos" == "windows" ]]; then
    ldflags="-H windowsgui ${ldflags}"
fi

# For Windows builds using MinGW, regenerate the icon resource for
# the target arch. For Zig-based builds (arm64), remove the pre-built
# .syso since it targets amd64 and no windres equivalent is available.
# For Linux builds, remove the .syso to avoid architecture mismatch
# at link time (Go links .syso files unconditionally).
if [[ "$goos" == "windows" ]]; then
    if [[ "${use_zig:-false}" == "true" ]]; then
        echo ">> Removing pre-built resource.syso for Zig-based build"
        rm -f cmd/subrelay/resource.syso
    elif command -v "${mingw_prefix}-windres" >/dev/null 2>&1; then
        echo ">> Generating icon resource for ${mingw_prefix}"
        (cd cmd/subrelay && rm -f resource.syso && \
            "${mingw_prefix}-windres" --target="${mingw_prefix}" -O coff \
            -o resource.syso resource.rc)
    else
        echo "warning: ${mingw_prefix}-windres not found, using existing .syso" >&2
    fi
elif [[ "$goos" == "linux" ]]; then
    rm -f cmd/subrelay/resource.syso
fi

# Determine output filename and extension.
ext=""
[[ "$goos" == "windows" ]] && ext=".exe"
output="build/subrelay-${version}-${goos}-${goarch}${ext}"

echo ">> Building ${target} with CC=${CC}"
go build \
    -tags "${build_tags}" \
    -ldflags "${ldflags}" \
    -o "$output" \
    ./cmd/subrelay

echo ">> Built ${output}"

# Package Linux artifacts when requested.
if [[ "$do_package" == "true" && "$goos" == "linux" ]]; then
    echo ">> Packaging Linux artifacts"
    chmod +x packaging/package-linux.sh
    packaging/package-linux.sh "$output" "$version" "$deb_arch" "build"
fi
