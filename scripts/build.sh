#!/bin/bash

# useQ AI Assistant - Build Script
# Quick build script for development and distribution

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR=${BUILD_DIR:-"./build"}
BINARY_NAME="useq"
PLATFORMS=${PLATFORMS:-"linux/amd64 darwin/amd64 darwin/arm64 windows/amd64"}

log_info() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_step() {
    echo -e "${BLUE}🔄 $1${NC}"
}

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════╗"
    echo "║          🤖 useQ AI Assistant          ║"
    echo "║             Build Script               ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""
}

check_prerequisites() {
    log_step "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    log_info "Go ${GO_VERSION} found"
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]] || [[ ! -d "cmd" ]]; then
        log_error "Please run this script from the project root directory"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

clean_build_dir() {
    log_step "Cleaning build directory..."
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"
    log_info "Build directory cleaned"
}

get_build_info() {
    # Get build information
    BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
    GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo '')
    
    if [[ -n "$GIT_TAG" ]]; then
        VERSION="$GIT_TAG"
    fi
    
    log_info "Version: $VERSION"
    log_info "Build time: $BUILD_TIME"
    log_info "Git commit: $GIT_COMMIT"
}

build_dependencies() {
    log_step "Downloading dependencies..."
    go mod tidy
    go mod download
    log_info "Dependencies downloaded"
}

build_single() {
    local goos="$1"
    local goarch="$2"
    local output_name="$3"
    
    log_step "Building for ${goos}/${goarch}..."
    
    # Set CGO_ENABLED based on platform
    local cgo_enabled=1
    if [[ "$goos" == "windows" ]] || [[ "$goarch" == "arm64" && "$goos" == "darwin" ]]; then
        cgo_enabled=0
    fi
    
    # Build flags
    local ldflags="-s -w"
    ldflags="$ldflags -X main.version=$VERSION"
    ldflags="$ldflags -X main.buildTime=$BUILD_TIME"
    ldflags="$ldflags -X main.gitCommit=$GIT_COMMIT"
    
    # Build
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED="$cgo_enabled" \
    go build -ldflags "$ldflags" -o "$BUILD_DIR/$output_name" cmd/main.go
    
    # Compress for release builds
    if [[ "${COMPRESS_BUILDS:-}" == "true" ]]; then
        if command -v upx &> /dev/null && [[ "$goos" != "darwin" ]]; then
            log_step "Compressing binary with UPX..."
            upx --best "$BUILD_DIR/$output_name" || log_warn "UPX compression failed"
        fi
    fi
    
    log_info "Built: $output_name"
}

build_all_platforms() {
    log_step "Building for all platforms..."
    
    for platform in $PLATFORMS; do
        local goos=$(echo $platform | cut -d'/' -f1)
        local goarch=$(echo $platform | cut -d'/' -f2)
        local extension=""
        
        if [[ "$goos" == "windows" ]]; then
            extension=".exe"
        fi
        
        local output_name="${BINARY_NAME}-${VERSION}-${goos}-${goarch}${extension}"
        build_single "$goos" "$goarch" "$output_name"
    done
    
    log_info "All platforms built successfully"
}

build_current_platform() {
    log_step "Building for current platform..."
    
    local goos=$(go env GOOS)
    local goarch=$(go env GOARCH)
    local extension=""
    
    if [[ "$goos" == "windows" ]]; then
        extension=".exe"
    fi
    
    build_single "$goos" "$goarch" "${BINARY_NAME}${extension}"
    
    # Create a symlink for easy access
    if [[ "$goos" != "windows" ]]; then
        ln -sf "${BINARY_NAME}" "$BUILD_DIR/useq-latest"
    fi
    
    log_info "Current platform build complete"
}

run_tests() {
    log_step "Running tests..."
    
    # Unit tests
    if go test -v ./... > "$BUILD_DIR/test-results.txt" 2>&1; then
        log_info "All tests passed"
    else
        log_warn "Some tests failed. Check $BUILD_DIR/test-results.txt"
    fi
    
    # Race condition tests
    if go test -race ./... > "$BUILD_DIR/race-test-results.txt" 2>&1; then
        log_info "Race condition tests passed"
    else
        log_warn "Race condition tests failed. Check $BUILD_DIR/race-test-results.txt"
    fi
}

lint_code() {
    log_step "Running code linting..."
    
    # Check if golangci-lint is available
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./... > "$BUILD_DIR/lint-results.txt" 2>&1 || {
            log_warn "Linting issues found. Check $BUILD_DIR/lint-results.txt"
        }
    else
        # Fallback to basic go vet and fmt
        go vet ./...
        if ! go fmt ./... | grep -q .; then
            log_info "Code formatting is correct"
        else
            log_warn "Code formatting issues found. Run: go fmt ./..."
        fi
    fi
}

create_checksums() {
    log_step "Creating checksums..."
    
    cd "$BUILD_DIR"
    for file in useq-*; do
        if [[ -f "$file" ]]; then
            sha256sum "$file" >> checksums.txt
            md5sum "$file" >> checksums.md5
        fi
    done
    cd - > /dev/null
    
    log_info "Checksums created"
}

create_archives() {
    log_step "Creating release archives..."
    
    cd "$BUILD_DIR"
    for file in useq-*; do
        if [[ -f "$file" && "$file" != *.* ]]; then
            # Create tar.gz for Unix platforms
            if [[ "$file" != *windows* ]]; then
                tar -czf "${file}.tar.gz" "$file"
                rm "$file"
            else
                # Create zip for Windows
                zip "${file}.zip" "$file"
                rm "$file"
            fi
        fi
    done
    cd - > /dev/null
    
    log_info "Release archives created"
}

show_build_summary() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════╗"
    echo -e "║           🎉 Build Complete! 🎉        ║"
    echo -e "╚════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}📦 Build artifacts:${NC}"
    ls -la "$BUILD_DIR"
    echo ""
    echo -e "${YELLOW}📊 Binary sizes:${NC}"
    du -h "$BUILD_DIR"/* | grep -E '\.(exe|tar\.gz|zip)$|useq$'
    echo ""
    if [[ -f "$BUILD_DIR/useq" ]]; then
        echo -e "${BLUE}🚀 Quick test:${NC}"
        echo "  $BUILD_DIR/useq --version"
        echo "  $BUILD_DIR/useq --help"
    fi
    echo ""
}

# Command line argument parsing
case "${1:-}" in
    --help|-h)
        echo "useQ AI Assistant Build Script"
        echo ""
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h          Show this help"
        echo "  --current           Build for current platform only (default)"
        echo "  --all               Build for all platforms"
        echo "  --release           Build release version with compression"
        echo "  --test              Run tests before building"
        echo "  --lint              Run linting before building"
        echo "  --clean             Clean build directory only"
        echo ""
        echo "Environment Variables:"
        echo "  VERSION             Set build version (default: 1.0.0)"
        echo "  BUILD_DIR           Set build directory (default: ./build)"
        echo "  PLATFORMS           Set platforms to build (space separated)"
        echo "  COMPRESS_BUILDS     Enable UPX compression (true/false)"
        echo ""
        echo "Examples:"
        echo "  $0                  # Build for current platform"
        echo "  $0 --all            # Build for all platforms"
        echo "  VERSION=2.0.0 $0    # Build specific version"
        echo ""
        exit 0
        ;;
    --clean)
        print_header
        clean_build_dir
        log_info "Build directory cleaned"
        exit 0
        ;;
    --all)
        BUILD_ALL=true
        ;;
    --release)
        BUILD_ALL=true
        COMPRESS_BUILDS=true
        RUN_TESTS=true
        CREATE_ARCHIVES=true
        ;;
    --test)
        RUN_TESTS=true
        ;;
    --lint)
        RUN_LINT=true
        ;;
    --current|"")
        BUILD_CURRENT=true
        ;;
    *)
        log_error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac

# Main build process
main() {
    print_header
    check_prerequisites
    get_build_info
    clean_build_dir
    build_dependencies
    
    # Optional steps
    if [[ "${RUN_LINT:-}" == "true" ]]; then
        lint_code
    fi
    
    if [[ "${RUN_TESTS:-}" == "true" ]]; then
        run_tests
    fi
    
    # Build step
    if [[ "${BUILD_ALL:-}" == "true" ]]; then
        build_all_platforms
        create_checksums
        if [[ "${CREATE_ARCHIVES:-}" == "true" ]]; then
            create_archives
        fi
    else
        build_current_platform
    fi
    
    show_build_summary
}

# Run main function
main