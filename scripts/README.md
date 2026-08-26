# scripts

## Purpose

Local build helper scripts for cross-compiling subrelay to all
supported Windows and Linux architectures from a single developer
machine.

## Contents

- `build.sh` - Shell script that builds subrelay for a single
  GOOS/GOARCH target with the correct CGO cross-compiler settings.
  Optionally produces Linux packages (.deb, .rpm, .tar.gz) when
  invoked with the --package flag.

## Dependencies and interactions

- `build.sh` calls `packaging/package-linux.sh` when the --package
  flag is used for Linux targets.
- The appropriate C cross-compiler must be installed for each
  target. See the table below for the required compiler per target.
- On Windows, MinGW-w64 must be on PATH. On Linux, install the
  cross-compiler packages listed below.

## Supported targets and required cross-compilers

| Target          | CC                          | Install (Debian/Ubuntu)                    |
| --------------- | --------------------------- | ------------------------------------------ |
| linux/amd64     | gcc                         | build-essential                            |
| linux/arm64     | aarch64-linux-gnu-gcc       | gcc-aarch64-linux-gnu                      |
| linux/arm       | arm-linux-gnueabihf-gcc     | gcc-arm-linux-gnueabihf                    |
| linux/386       | gcc -m32                    | gcc-multilib                               |
| windows/amd64   | x86_64-w64-mingw32-gcc      | gcc-mingw-w64-x86-64                       |
| windows/386     | i686-w64-mingw32-gcc        | gcc-mingw-w64-i686                         |
| windows/arm64   | aarch64-w64-mingw32-gcc     | gcc-mingw-w64-arm64                        |

## Usage examples

```sh
# Build for the host platform
scripts/build.sh linux/amd64 1.0.0

# Build and package Linux .deb/.rpm/.tar.gz
scripts/build.sh linux/arm64 1.0.0 --package

# Cross-compile Windows .exe from Linux
scripts/build.sh windows/amd64 1.0.0
```
