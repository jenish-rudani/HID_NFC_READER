BIN_DIR := bin/
BIN_NAME_PREFIX := hidnfcreader
CODE_ENTRY := .

# Version information
VERSION_FILE := version.go
VERSION := $(shell grep -E "VERSION.*=.*\".*\"" ${VERSION_FILE} | cut -d '"' -f 2)
GIT_COMMIT := $(shell git rev-list -1 HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
NEW_VERSION := $(shell git describe --tags --abbrev=0)

# Get OneDrive paths from environment
ONEDRIVE_WINDOWS := $(shell echo $$ONE_DRIVE_TOOLS_WINDOWS)
ONEDRIVE_MAC := $(shell echo $$ONE_DRIVE_TOOLS_MAC)

# LDFLAGS for version information
LDFLAGS := -ldflags="\
	-X 'main.VERSION=${NEW_VERSION}' \
	-X 'main.GITCOMMIT=${GIT_COMMIT}' \
	-X 'main.BUILDTIME=${BUILD_TIME}' \
	-s -w"

# Add version commands
.PHONY: version bump-patch bump-minor bump-major

version:
	@echo "Current version: ${VERSION}"
	@echo "Git commit: ${GIT_COMMIT}"
	@echo "Build time: ${BUILD_TIME}"

all: clean compile-all

compile-all: compile-darwin compile-windows

compile-darwin:
	@echo "Building for macOS..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BIN_DIR}mac/${BIN_NAME_PREFIX}_arm64 ${CODE_ENTRY}
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BIN_DIR}mac/${BIN_NAME_PREFIX}_amd64 ${CODE_ENTRY}
	lipo -create -output ${BIN_DIR}mac/${BIN_NAME_PREFIX}_${NEW_VERSION} ${BIN_DIR}mac/${BIN_NAME_PREFIX}_arm64 ${BIN_DIR}mac/${BIN_NAME_PREFIX}_amd64
	rm ${BIN_DIR}mac/${BIN_NAME_PREFIX}_arm64 ${BIN_DIR}mac/${BIN_NAME_PREFIX}_amd64
	@if [ ! -z "${ONEDRIVE_MAC}" ]; then \
		echo "Copying to OneDrive Mac folder..."; \
		cp ${BIN_DIR}mac/${BIN_NAME_PREFIX}_${NEW_VERSION} "${ONEDRIVE_MAC}/${BIN_NAME_PREFIX}_${NEW_VERSION}"; \
	fi
	@echo "macOS build complete"

update-version:
	@echo "Bumping major version..."
	@echo "Version bumped to: ${NEW_VERSION}"
	sed -i '' "s/VERSION = \"[^\"]*\"/VERSION = "\"${NEW_VERSION}\""/" version.go

compile-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build ${LDFLAGS} -o ${BIN_DIR}windows/${BIN_NAME_PREFIX}_${NEW_VERSION}.exe ${CODE_ENTRY}
	@if [ ! -z "${ONEDRIVE_WINDOWS}" ]; then \
		echo "Copying to OneDrive Windows folder..."; \
		cp ${BIN_DIR}windows/${BIN_NAME_PREFIX}_${NEW_VERSION}.exe "${ONEDRIVE_WINDOWS}/${BIN_NAME_PREFIX}_${NEW_VERSION}.exe"; \
	fi
	@echo "Windows build complete"

compile-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BIN_DIR}linux/${BIN_NAME_PREFIX} ${CODE_ENTRY}
	@echo "Linux build complete"

clean:
	@echo "Cleaning..."
	rm -rf ${BIN_DIR}

# Development helpers
run-dev:
	go run ${CODE_ENTRY}

tidy:
	go mod tidy
	go mod vendor

# Release packaging
release: compile-all
	@echo "Creating release package..."
	rm -f ${BIN_NAME_PREFIX}-release-*.zip
	zip -vr ${BIN_NAME_PREFIX}-release-${GIT_COMMIT}.zip ${BIN_DIR} -x "*.DS_Store"
	@echo "Release package created"

# Git checks
git-porcelain:
	@echo "Commit: ${GIT_COMMIT}"
	@status=$$(git status --porcelain); \
	if [ ! -z "$${status}" ]; \
	then \
	   echo "Error - working directory is dirty. Commit those changes!"; \
	   exit 1; \
	fi

# Full release process
release-all: git-porcelain clean compile-all
	@echo "Creating full release..."
	rm -f ${BIN_NAME_PREFIX}-release-*.zip
	zip -vr ${BIN_NAME_PREFIX}-release-${GIT_COMMIT}.zip ${BIN_DIR} -x "*.DS_Store"
	@echo "Full release complete"