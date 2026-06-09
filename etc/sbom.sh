#!/usr/bin/env bash

set -o errexit
set -o pipefail

: "${branch_name:?}"
: "${DOCKER_CONFIG:?}"
: "${AWS_ACCESS_KEY_ID:?}"
: "${AWS_SECRET_ACCESS_KEY:?}"

command -v podman >/dev/null || {
  echo "missing required program podman" 1>&2
  exit 1
}

command -v jq >/dev/null || {
  echo "missing required program jq" 1>&2
  exit 1
}

silkbomb="901841024863.dkr.ecr.us-east-1.amazonaws.com/release-infrastructure/silkbomb:2.0"

podman pull "${silkbomb:?}"

silkbomb_augment_flags=(
  --repo mongodb/mongo-go-driver
  --branch "${branch_name:?}"
  --sbom-in /pwd/sbom.json
  --sbom-out /pwd/augmented.sbom.json.new

  # Any notable updates to the Augmented SBOM version should be done manually after careful inspection.
  # Otherwise, it should be equal to the existing SBOM version.
  --no-update-sbom-version
)

podman run -it --rm -v "$(pwd):/pwd" --env 'AWS_ACCESS_KEY_ID' --env 'AWS_SECRET_ACCESS_KEY' \
  "${silkbomb:?}" augment "${silkbomb_augment_flags[@]:?}"

[[ -f ./augmented.sbom.json.new ]] || {
  echo "failed to download Augmented SBOM" 1>&2
  exit 1
}

echo "Comparing Augmented SBOM..."

jq -S 'del(.metadata.timestamp)' ./augmented.sbom.json >|old.json
jq -S 'del(.metadata.timestamp)' ./augmented.sbom.json.new >|new.json

if ! diff -sty --left-column -W 200 old.json new.json >|diff.txt; then
  declare status
  status='{"status":"failed", "type":"test", "should_continue":true, "desc":"detected significant changes in Augmented SBOM"}'
  curl -sS -d "${status:?}" -H "Content-Type: application/json" -X POST localhost:2285/task_status || true
fi

cat diff.txt

echo "Comparing Augmented SBOM... done."
