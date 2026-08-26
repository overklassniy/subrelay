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
- The workflow installs MinGW cross-compilers for Windows targets
  and gcc cross-toolchains with multiarch dev libraries for Linux
  targets.
- On tag pushes, a GitHub Release is created with all artifacts
  attached via `softprops/action-gh-release`.
