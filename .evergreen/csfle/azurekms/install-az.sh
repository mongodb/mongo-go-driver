#!/usr/bin/env bash
set -o errexit
set -o pipefail
set -o nounset

# Install az CLI for Debian/Ubuntu.
# https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-linux?pivots=apt#option-2-step-by-step-installation-instructions
echo "Install az ... begin"
if command -v az &> /dev/null; then
    echo "removing old az package ..."
    sudo apt remove azure-cli -y && sudo apt autoremove -y
fi
sudo apt-get update
sudo apt-get install -y ca-certificates curl apt-transport-https lsb-release gnupg
curl -sL https://packages.microsoft.com/keys/microsoft.asc |
    gpg --dearmor |
    sudo tee /etc/apt/trusted.gpg.d/microsoft.gpg > /dev/null
AZ_REPO=$(lsb_release -cs)
echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $AZ_REPO main" |
    sudo tee /etc/apt/sources.list.d/azure-cli.list
sudo apt-get update
sudo apt-get install -y azure-cli
echo "Install az ... end"
