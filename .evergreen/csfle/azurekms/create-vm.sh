#!/usr/bin/env bash
set -o errexit
set -o pipefail
set -o nounset

# Create an Azure VM. `az` is expected to be logged in.
if [ -z "${AZUREKMS_VMNAME_PREFIX:-}" ] || \
   [ -z "${AZUREKMS_RESOURCEGROUP:-}" ] || \
   [ -z "${AZUREKMS_IMAGE:-}" ] || \
   [ -z "${AZUREKMS_PUBLICKEYPATH:-}" ]; then
    echo "Please set the following required environment variables"
    echo " AZUREKMS_VMNAME_PREFIX to an identifier string no spaces (e.g. CDRIVER)"
    echo " AZUREKMS_RESOURCEGROUP"
    echo " AZUREKMS_IMAGE to the image (e.g. Debian)"
    echo " AZUREKMS_PUBLICKEYPATH to the path to the public SSH key"
    exit 1
fi

AZUREKMS_VMNAME="vmname-$AZUREKMS_VMNAME_PREFIX-$RANDOM"
echo "Creating a Virtual Machine ($AZUREKMS_VMNAME) ... begin"
# az vm create also creates a "nic" and "public IP" by default.
# Use --nic-delete-option 'Delete' to delete the NIC.
# Specify a name for the public IP to delete later.
# Pipe to /dev/null to hide the output. The output includes tenantId.
az vm create \
    --resource-group "$AZUREKMS_RESOURCEGROUP" \
    --name "$AZUREKMS_VMNAME" \
    --image "$AZUREKMS_IMAGE" \
    --admin-username azureuser \
    --ssh-key-values "$AZUREKMS_PUBLICKEYPATH" \
    --public-ip-sku Standard \
    --nic-delete-option delete \
    --data-disk-delete-option delete \
    --os-disk-delete-option delete \
    --public-ip-address "$AZUREKMS_VMNAME-PUBLIC-IP" \
    --assign-identity "[system]" \
    >/dev/null
echo "Creating a Virtual Machine ($AZUREKMS_VMNAME) ... end"