set shell := ["bash", "-eu", "-o", "pipefail", "-c"]
set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-eu", "-o", "pipefail", "-c"]

# Verify required source files still exist before running broader gates.
[private]
verify-sources:
  @git ls-files --error-unmatch cmd/till/main.go cmd/till/main_test.go >/dev/null

# Format all tracked Go files in place.
fmt:
  @set -- $(git ls-files '*.go'); \
  if [ "$#" -gt 0 ]; then \
    gofmt -w "$@"; \
  fi

# Verify all tracked Go files are already gofmt-formatted.
[private]
fmt-check:
  @set -- $(git ls-files '*.go'); \
  if [ "$#" -eq 0 ]; then \
    exit 0; \
  fi; \
  out="$(gofmt -l "$@")"; \
  if [ -n "$out" ]; then \
    echo "gofmt required for:"; \
    echo "$out"; \
    exit 1; \
  fi

# Run the full repository Go test suite.
test:
  @go test ./...

# Run tests for a specific package path, directory, or pattern.
test-pkg pkg:
  @pkg="{{pkg}}"; \
  if [ -d "$pkg" ]; then \
    if ls "$pkg"/*.go >/dev/null 2>&1; then \
      go test "$pkg"; \
    else \
      go test "$pkg/..."; \
    fi; \
  else \
    go test "$pkg"; \
  fi

# Run golden-file tests for the TUI package.
test-golden:
  @go test ./internal/tui -run 'Golden'

# Update and re-run golden-file tests for the TUI package.
test-golden-update:
  @go test ./internal/tui -run 'Golden' -update

# Build the local till binary at ./till.
build:
  @go build -o ./till ./cmd/till

# Run till directly from source.
run:
  @go run ./cmd/till

# Initialize a dev config file at the resolved --dev config path if missing.
init-dev-config:
  @./till --dev init-dev-config

# Delete the resolved --dev runtime root (db/config/logs under that root).
clean-dev:
  @root_dir="$(./till --dev paths | awk -F': ' '/^root:/{print $2}')"; \
  if [ -z "$root_dir" ]; then \
    echo "could not resolve dev runtime root"; \
    exit 1; \
  fi; \
  rm -rf "$root_dir"; \
  echo "removed dev runtime root: $root_dir"

# Run VHS for a single tape or all tapes under vhs/.
vhs tape="":
  @mkdir -p .artifacts/vhs; \
  if [ -n "{{tape}}" ]; then \
    vhs "{{tape}}"; \
  else \
    for t in vhs/*.tape; do \
      vhs "$t"; \
    done; \
  fi

# Enforce per-package coverage floor on the full test run output.
[private]
coverage:
  @tmp=$(mktemp); \
  go test ./... -cover | tee "$tmp"; \
  awk 'BEGIN {bad=0} \
    /^ok[[:space:]]/ && /coverage:/ { \
      covLine=$0; \
      sub(/^.*coverage:[[:space:]]*/, "", covLine); \
      sub(/%.*/, "", covLine); \
      cov=covLine+0; \
      if (cov < 70) { \
        print "coverage below 70%:", $2, covLine "%"; \
        bad=1; \
      } \
    } \
    END {exit bad}' "$tmp"; \
  rm -f "$tmp"

# Cross-platform smoke gate: source verification, formatting, tests, and build.
check: verify-sources fmt-check test build

# Canonical full gate: smoke gate + coverage floor enforcement.
ci: verify-sources fmt-check coverage build
