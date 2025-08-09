#!/usr/bin/env bash
set -e

CHECK_CURRENCY="false"

# Options are:
# -c : check currency of staged sbom.json versus go.mod.
while getopts "c" opt; do
    case $opt in
        c)
            CHECK_CURRENCY="true"
            ;;
        *)
            echo "usage: $0 [-c]" >&2
            echo " -c : (optional) check currency of staged sbom.json versus go.mod." >&2
            exit 1
            ;;
    esac
done
#shift $((OPTIND - 1))

if  ! $CHECK_CURRENCY; then
    # The cyclonedx-gomod 'mod' subcommand is used to generate a CycloneDX SBOM with GOWORK=off to exclude example/test code.
    # TODO: Add libmongocrypt as an optional component via a merge once the libmongocrypt SBOM is updated with newer automation

    ## The pipe to jq is a temporary workaround until this issue is resolved: https://github.com/CycloneDX/cyclonedx-gomod/issues/662.
    ## When resolved, bump version and replace with commented line below.
    # GOWORK=off go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@[UPDATED VERSION] mod -type library -licenses -assert-licenses -output-version 1.5 -json -output sbom.json .
    GOWORK=off go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.9.0 mod -type library -licenses -assert-licenses -output-version 1.5 -json . | jq '.metadata.component.purl |= split("?")[0]' | jq '.components[].purl |= split("?")[0]' > sbom.json
elif [[ $(git diff --name-only --cached go.mod) && ! $(git diff --name-only --cached sbom.json) ]]; then
    echo "'go.mod' has changed. 'sbom.json' must be re-generated (run 'task generate-sbom' or 'etc/generate-sbom.sh') and staged." && exit 1
fi
