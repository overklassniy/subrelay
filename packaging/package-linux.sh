#!/bin/bash
# package-linux.sh assembles .deb, .rpm, and .tar.gz packages
# from a pre-built subrelay binary.
#
# The script creates a temporary staging directory with the
# filesystem layout expected by Debian and RPM packages:
#
#   usr/local/bin/subrelay          - the executable
#   usr/share/applications/         - the .desktop menu entry
#   usr/share/pixmaps/subrelay.png  - the application icon
#
# It then builds a .deb using dpkg-deb, a .rpm using rpmbuild
# (or fpm when available), and a .tar.gz archive.
#
# Usage:
#   package-linux.sh <binary-path> <version> <deb-arch> <output-dir>
#
# Args:
#   - binary-path: path to the pre-built subrelay binary.
#   - version: semantic version string (e.g. 1.0.0).
#   - deb-arch: Debian architecture name (amd64, arm64, armhf, i386).
#   - output-dir: directory where packages are written.
#
# Errors:
#   - Exits with a non-zero status when the binary is missing,
#     a required tool is unavailable, or packaging fails.
set -euo pipefail

binary_path="${1:?usage: package-linux.sh <binary> <version> <deb-arch> <output-dir>}"
version="${2:?usage: package-linux.sh <binary> <version> <deb-arch> <output-dir>}"
deb_arch="${3:?usage: package-linux.sh <binary> <version> <deb-arch> <output-dir>}"
output_dir="${4:?usage: package-linux.sh <binary> <version> <deb-arch> <output-dir>}"

script_dir="$(cd "$(dirname "$0")" && pwd)"
project_root="$(cd "$script_dir/.." && pwd)"
icon_source="$project_root/internal/tray/icon.png"
desktop_source="$script_dir/linux/subrelay.desktop"
control_template="$script_dir/linux/control.template"
postinst_source="$script_dir/linux/postinst"
postrm_source="$script_dir/linux/postrm"

if [[ ! -f "$binary_path" ]]; then
    echo "error: binary not found: $binary_path" >&2
    exit 1
fi

if [[ ! -f "$icon_source" ]]; then
    echo "error: icon not found: $icon_source" >&2
    exit 1
fi

mkdir -p "$output_dir"

# Staging directory for the package filesystem layout.
staging="$(mktemp -d)"
trap 'rm -rf "$staging"' EXIT

# Populate the staging directory.
install -Dm755 "$binary_path" "$staging/usr/local/bin/subrelay"
install -Dm644 "$desktop_source" "$staging/usr/share/applications/subrelay.desktop"
install -Dm644 "$icon_source" "$staging/usr/share/pixmaps/subrelay.png"

# Compute installed size in kilobytes for the control file.
installed_size_kb="$(du -sk "$staging" | cut -f1)"

# --- .tar.gz ---
tarball="$output_dir/subrelay-${version}-linux-${deb_arch}.tar.gz"
tar -czf "$tarball" -C "$staging" .
echo "wrote $tarball"

# --- .deb ---
deb_root="$(mktemp -d)"
trap 'rm -rf "$staging" "$deb_root"' EXIT

mkdir -p "$deb_root/DEBIAN"
cp -a "$staging/." "$deb_root/"

# Generate the control file from the template.
sed \
    -e "s/__VERSION__/${version}/g" \
    -e "s/__ARCH__/${deb_arch}/g" \
    -e "s/__INSTALLED_SIZE__/${installed_size_kb}/g" \
    "$control_template" > "$deb_root/DEBIAN/control"

install -Dm755 "$postinst_source" "$deb_root/DEBIAN/postinst"
install -Dm755 "$postrm_source" "$deb_root/DEBIAN/postrm"

deb_file="$output_dir/subrelay-${version}-linux-${deb_arch}.deb"
dpkg-deb --build --root-owner-group "$deb_root" "$deb_file"
echo "wrote $deb_file"

# --- .rpm ---
rpm_file="$output_dir/subrelay-${version}-linux-${deb_arch}.rpm"
if command -v fpm >/dev/null 2>&1; then
    # Map Debian arch names to RPM arch names.
    case "$deb_arch" in
        amd64)  rpm_arch="x86_64" ;;
        arm64)  rpm_arch="aarch64" ;;
        armhf)  rpm_arch="armv7hl" ;;
        i386)   rpm_arch="i686" ;;
        *)      rpm_arch="$deb_arch" ;;
    esac

    fpm \
        --input-type dir \
        --output-type rpm \
        --name subrelay \
        --version "$version" \
        --architecture "$rpm_arch" \
        --maintainer "overklassniy <overklassniy@users.noreply.github.com>" \
        --description "Subscription-based proxy balancer with system tray control" \
        --url "https://github.com/overklassniy/subrelay" \
        --depends "libgl1" \
        --depends "libegl" \
        --depends "libxkbcommon" \
        --chdir "$staging" \
        --package "$rpm_file" \
        .
    echo "wrote $rpm_file"
else
    echo "warning: fpm not found, skipping .rpm package" >&2
fi
