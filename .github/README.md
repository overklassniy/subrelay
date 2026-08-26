# .github

## Purpose

GitHub-specific configuration, including CI/CD workflow definitions.

## Contents

- `workflows/release.yml` - Release workflow that builds subrelay
  for all supported Windows and Linux architectures and publishes
  distributable artifacts. Triggered on version tag pushes (v*) or
  manual dispatch. Windows targets produce .zip files containing
  the .exe; Linux targets produce .deb, .rpm, and .tar.gz packages.

## Dependencies and interactions

- The workflow uses `packaging/package-linux.sh` to assemble Linux
  packages after each build.
- The workflow installs MinGW cross-compilers for Windows amd64 and
  386 targets. Windows arm64 uses Zig as a drop-in C cross-compiler
  since Ubuntu does not ship an aarch64-w64-mingw32 MinGW toolchain.
- The workflow installs gcc cross-toolchains with multiarch dev
  libraries for Linux targets. For arm64 and armhf, apt sources are
  configured to use ports.ubuntu.com since the default Ubuntu archive
  does not carry foreign-architecture package indices.
- The committed `cmd/subrelay/resource.syso` (an amd64 Windows COFF
  object) is removed before Linux builds and Windows arm64 builds to
  avoid architecture mismatch at link time.
- On tag pushes, a GitHub Release is created with all artifacts
  attached via `softprops/action-gh-release`.
