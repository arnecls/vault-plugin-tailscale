default:
    @just -l

# Create a release from the latest tag
release:
    #!/usr/bin/env bash
    set -euo pipefail

    export GITHUB_TOKEN="$(gh auth token)"
    goreleaser release --config {{justfile_directory()}}/.goreleaser.yml --clean

