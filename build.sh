#!/usr/bin/env bash
# Build script for Blaze cross-platform package manager
# Builds binaries for Linux (amd64 and arm64)

set -o pipefail

OUTPUT_DIR="dist"
CLEAN=0
VERBOSE=0

usage() {
  cat <<'EOF'
Usage: build.sh [options]

Options:
  -o, --output-dir DIR   Output directory (default: dist)
  -c, --clean            Remove the output directory before building
  -v, --verbose          Print the go build command for each target
  -h, --help             Show this help message
EOF
}

write_success() {
  printf '%s\n' "$*"
}

write_error() {
  printf '%s\n' "$*" >&2
}

write_info() {
  printf '%s\n' "$*"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--output-dir)
      if [[ $# -lt 2 ]]; then
        write_error "Missing value for $1"
        exit 1
      fi
      OUTPUT_DIR="$2"
      shift 2
      ;;
    -c|--clean)
      CLEAN=1
      shift
      ;;
    -v|--verbose)
      VERBOSE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      write_error "Unknown argument: $1"
      usage >&2
      exit 1
      ;;
  esac
done

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir"

if [[ $CLEAN -eq 1 || ! -d "$OUTPUT_DIR" ]]; then
  write_info "Cleaning output directory..."
  if [[ -d "$OUTPUT_DIR" ]]; then
    rm -rf "$OUTPUT_DIR"
  fi
  mkdir -p "$OUTPUT_DIR"
fi

write_info "Building Blaze binaries..."
write_info "Output directory: $OUTPUT_DIR"

targets=(
  "linux amd64 blaze-linux-amd64"
  "linux arm64 blaze-linux-arm64"
)

successful=0
failed=0

for target in "${targets[@]}"; do
  read -r os arch output_name <<< "$target"
  output_file="$OUTPUT_DIR/blaze-${os}-${arch}/${output_name}"
  output_folder="$(dirname "$output_file")"

  write_info "Building for $os/$arch..."

  mkdir -p "$output_folder"

  if [[ $VERBOSE -eq 1 ]]; then
    write_info "  Command: GOOS=$os GOARCH=$arch go build -o $output_file ./src"
  fi

  build_output="$(GOOS="$os" GOARCH="$arch" go build -o "$output_file" ./src 2>&1)"

  if [[ $? -eq 0 ]]; then
    if command -v stat >/dev/null 2>&1; then
      if stat --version >/dev/null 2>&1; then
        file_size_bytes="$(stat -c '%s' "$output_file")"
      else
        file_size_bytes="$(stat -f '%z' "$output_file")"
      fi
      file_size_mb="$(awk -v size="$file_size_bytes" 'BEGIN { printf "%.2f", size / (1024 * 1024) }')"
      write_success "  OK - $(basename "$output_file") (${file_size_mb} MB)"
    else
      write_success "  OK - $(basename "$output_file")"
    fi
    successful=$((successful + 1))
  else
    write_error "  FAILED - $build_output"
    failed=$((failed + 1))
  fi
done

write_info ""
write_info "Build Summary:"
write_success "  Successful: $successful"
if [[ $failed -gt 0 ]]; then
  write_error "  Failed: $failed"
else
  write_success "  Failed: $failed"
fi

if [[ $failed -eq 0 ]]; then
  write_success ""
  write_success "All builds completed successfully!"
  write_info "Binaries available in: $OUTPUT_DIR"
  write_info ""
  write_info "Next steps:"
  write_info "  - Test: ./dist/blaze-linux-amd64/blaze-linux-amd64 list"
  write_info "  - Package and distribute binaries"
else
  exit 1
fi
