#!/usr/bin/env bash
set -eo pipefail

# The cyclonedx-gomod 'mod' subcommand is used to generate a CycloneDX SBOM with GOWORK=off to exclude example/test code.

## The pipe to jq is a temporary workaround until this issue is resolved: https://github.com/CycloneDX/cyclonedx-gomod/issues/662.
## When resolved, bump version and replace with commented line below.
# GOWORK=off go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@[UPDATED VERSION] mod -type library -licenses -assert-licenses -output-version 1.5 -json -output sbom.json .
GOWORK=off go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0 mod -type library -licenses -assert-licenses -output-version 1.5 -json . | jq '.metadata.component.purl |= split("?")[0]' | jq '.components[].purl |= split("?")[0]' > sbom.json

# Derive the libmongocrypt version from the install script and inject it as an optional component.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIBMONGOCRYPT_VERSION=$(grep 'LIBMONGOCRYPT_TAG=' "${SCRIPT_DIR}/install-libmongocrypt.sh" | head -1 | cut -d'"' -f2)
LIBMONGOCRYPT_PURL="pkg:github/mongodb/libmongocrypt@${LIBMONGOCRYPT_VERSION}"

jq --arg version "$LIBMONGOCRYPT_VERSION" --arg purl "$LIBMONGOCRYPT_PURL" '
  .metadata.component."bom-ref" as $main_ref |
  .components += [{
    "type": "library",
    "bom-ref": $purl,
    "supplier": {"name": "MongoDB, Inc.", "url": ["https://mongodb.com"]},
    "author": "MongoDB, Inc.",
    "group": "mongodb",
    "name": "libmongocrypt",
    "version": $version,
    "description": "Required C library for Client Side and Queryable Encryption in MongoDB",
    "scope": "optional",
    "licenses": [{"license": {"id": "Apache-2.0"}}],
    "copyright": "Copyright 2019-present MongoDB, Inc.",
    "cpe": ("cpe:2.3:a:mongodb:libmongocrypt:" + $version + ":*:*:*:*:*:*:*"),
    "purl": $purl,
    "externalReferences": [{"url": "https://github.com/mongodb/libmongocrypt.git", "type": "distribution"}]
  }] |
  (.dependencies[] | select(.ref == $main_ref) | .dependsOn) += [$purl] |
  .dependencies += [{"ref": $purl, "dependsOn": []}]
' sbom.json > sbom.tmp.json && mv sbom.tmp.json sbom.json
