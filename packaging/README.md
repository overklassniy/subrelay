# packaging

## Purpose

Contains packaging templates and scripts used by the CI workflow
and local build scripts to produce distributable artifacts (.deb,
.rpm, .tar.gz for Linux; .zip for Windows) from the compiled
subrelay binary.

## Contents

- `linux/control.template` - Debian control file template with
  placeholders (__VERSION__, __ARCH__, __INSTALLED_SIZE__) that are
  substituted at package build time.
- `linux/subrelay.desktop` - Freedesktop.org .desktop menu entry
  installed to /usr/share/applications on Linux.
- `linux/postinst` - Debian post-installation script that refreshes
  the desktop and icon databases.
- `linux/postrm` - Debian post-removal script that refreshes the
  desktop database.
- `package-linux.sh` - Shell script that assembles .deb, .rpm, and
  .tar.gz packages from a pre-built binary. Used by the CI workflow
  and the local Makefile.

## Dependencies and interactions

- `package-linux.sh` expects the binary to already be built for the
  target Linux architecture (GOOS=linux, GOARCH=<arch>).
- The icon is sourced from `internal/tray/icon.png`.
- The .desktop file references the `subrelay` icon name, which is
  installed as `/usr/share/pixmaps/subrelay.png`.
- .rpm packaging requires `fpm` (Effing Package Management) to be
  installed; .deb packaging requires `dpkg-deb`.
- The CI workflow (`.github/workflows/release.yml`) calls
  `package-linux.sh` after each Linux build job completes.
