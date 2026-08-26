# Makefile for local subrelay builds.
#
# Targets:
#   build            - Build for the host platform (native).
#   build-windows    - Build all Windows architectures.
#   build-linux      - Build all Linux architectures.
#   build-all        - Build every supported target.
#   package-linux    - Build and package all Linux targets (.deb/.rpm/.tar.gz).
#   test             - Run go test on all packages.
#   vet              - Run go vet on all packages.
#   fmt              - Check formatting with gofmt.
#   fmt-write        - Apply gofmt to all Go files.
#   clean            - Remove the build directory.
#   icon             - Regenerate the Windows icon resource (.syso).
#
# Override VERSION to inject a version string:
#   make build VERSION=1.2.3
#
# Build tags required by the sing-box extended fork.
BUILD_TAGS := with_utls,with_grpc,with_quic
VERSION ?= dev
BUILDDIR := build

.PHONY: build build-windows build-linux build-all package-linux test vet fmt fmt-write clean icon

# Native build for the host platform.
build:
	mkdir -p $(BUILDDIR)
	CGO_ENABLED=1 go build \
		-tags "$(BUILD_TAGS)" \
		-ldflags "-s -w -X subrelay/internal/update.CurrentVersion=$(VERSION)" \
		-o $(BUILDDIR)/subrelay \
		./cmd/subrelay

# Build all Windows architectures.
build-windows: build-windows-amd64 build-windows-386 build-windows-arm64

build-windows-%: GOOS := windows
build-windows-%: CGO_ENABLED := 1
build-windows-%:
	@target="$*"; \
	case "$$target" in \
		amd64) cc="x86_64-w64-mingw32-gcc"; prefix="x86_64-w64-mingw32" ;; \
		386)   cc="i686-w64-mingw32-gcc";   prefix="i686-w64-mingw32" ;; \
		arm64) cc="aarch64-w64-mingw32-gcc"; prefix="aarch64-w64-mingw32" ;; \
		*) echo "unsupported: $$target"; exit 1 ;; \
	esac; \
	mkdir -p $(BUILDDIR); \
	if command -v "$${prefix}-windres" >/dev/null 2>&1; then \
		(cd cmd/subrelay && rm -f resource.syso && \
			"$${prefix}-windres" --target="$${prefix}" -O coff -o resource.syso resource.rc); \
	fi; \
	GOOS=windows GOARCH=$$target CGO_ENABLED=1 CC="$$cc" CXX="$${cc/gcc/g++}" \
	go build -tags "$(BUILD_TAGS)" \
		-ldflags "-H windowsgui -s -w -X subrelay/internal/update.CurrentVersion=$(VERSION)" \
		-o $(BUILDDIR)/subrelay-$(VERSION)-windows-$$target.exe \
		./cmd/subrelay

# Build all Linux architectures.
build-linux: build-linux-amd64 build-linux-arm64 build-linux-arm build-linux-386

build-linux-%:
	@target="$*"; \
	case "$$target" in \
		amd64) cc="gcc"; deb_arch="amd64"; flags="" ;; \
		arm64) cc="aarch64-linux-gnu-gcc"; deb_arch="arm64"; flags="PKG_CONFIG_PATH=/usr/lib/aarch64-linux-gnu/pkgconfig" ;; \
		arm)   cc="arm-linux-gnueabihf-gcc"; deb_arch="armhf"; flags="GOARM=7 PKG_CONFIG_PATH=/usr/lib/arm-linux-gnueabihf/pkgconfig" ;; \
		386)   cc="gcc"; deb_arch="i386"; flags="CGO_CFLAGS=-m32 CGO_LDFLAGS=-m32 PKG_CONFIG_PATH=/usr/lib/i386-linux-gnu/pkgconfig" ;; \
		*) echo "unsupported: $$target"; exit 1 ;; \
	esac; \
	mkdir -p $(BUILDDIR); \
	GOOS=linux GOARCH=$$target CGO_ENABLED=1 CC="$$cc" CXX="$${cc/gcc/g++}" $$flags \
	go build -tags "$(BUILD_TAGS)" \
		-ldflags "-s -w -X subrelay/internal/update.CurrentVersion=$(VERSION)" \
		-o $(BUILDDIR)/subrelay-$(VERSION)-linux-$$target \
		./cmd/subrelay

# Build and package all Linux targets.
package-linux:
	@for target in amd64 arm64 arm 386; do \
		$(MAKE) build-linux-$$target VERSION=$(VERSION); \
		case "$$target" in \
			amd64) deb_arch="amd64" ;; \
			arm64) deb_arch="arm64" ;; \
			arm)   deb_arch="armhf" ;; \
			386)   deb_arch="i386" ;; \
		esac; \
		chmod +x packaging/package-linux.sh; \
		packaging/package-linux.sh \
			$(BUILDDIR)/subrelay-$(VERSION)-linux-$$target \
			$(VERSION) $$deb_arch $(BUILDDIR); \
	done

# Build everything.
build-all: build-windows build-linux

test:
	CGO_ENABLED=1 go test ./...

vet:
	CGO_ENABLED=1 go vet ./...

fmt:
	gofmt -l .

fmt-write:
	gofmt -w .

clean:
	rm -rf $(BUILDDIR)

# Regenerate the Windows icon resource from the tray PNG.
icon:
	cd cmd/subrelay && go run make_icon.go
