#!/usr/bin/env bash
set -eo pipefail

SERIAL_NUMBER="urn:uuid:b7adcdf8-bafc-43c5-a529-a73130697171"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate the base SBOM. -short-purls strips ?type=module query strings from PURLs,
# resolving https://github.com/CycloneDX/cyclonedx-gomod/issues/662.
GOWORK=off go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0 \
    mod -type library -licenses -assert-licenses -output-version 1.5 -json \
    -serial "$SERIAL_NUMBER" -std=true -short-purls=true . > sbom.tmp.json

# Inject libmongocrypt as an optional component.
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
' sbom.tmp.json > sbom.new.json
rm sbom.tmp.json

# Increment .version only when meaningful content has changed.
CURRENT_VERSION=$(jq -r '.version // 0' sbom.json 2>/dev/null || echo 0)
NEW_CONTENT=$(jq -S 'del(.version, .metadata.timestamp)' sbom.new.json)
OLD_CONTENT=$(jq -S 'del(.version, .metadata.timestamp)' sbom.json 2>/dev/null || echo '{}')

if [ "$NEW_CONTENT" = "$OLD_CONTENT" ]; then
    NEW_VERSION=$CURRENT_VERSION
else
    NEW_VERSION=$((CURRENT_VERSION + 1))
fi

jq --argjson v "$NEW_VERSION" '.version = $v' sbom.new.json > sbom.json
rm sbom.new.json
