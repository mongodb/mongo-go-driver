#!/usr/bin/env bash
# Runs an AWS SSO login, then writes the resulting credentials to
# $SECRETS_DIR/secrets-export.sh in the same "export KEY=VALUE" format used by
# drivers-evergreen-tools.
#
# "aws sso login --no-browser" prints a URL and a verification code to stdout;
# the caller is responsible for surfacing those so a human can complete the
# flow. ExportSecrets streams container logs to the test log for that reason.
set -eu

# The drivers test-secrets role. These are the values "aws configure sso"
# would otherwise prompt for, so hardcoding them removes the one-time
# interactive setup: a fresh checkout on a fresh machine can log in directly.
# None of them are secrets — they only identify which account and role the
# login should assume, which still requires an approved SSO session.
#
# Override any of them from the environment to log in somewhere else.
AWS_PROFILE=${AWS_PROFILE:-drivers-test-secrets-role-857654397073}
SSO_START_URL=${SSO_START_URL:-https://d-9067613a84.awsapps.com/start#}
SSO_REGION=${SSO_REGION:-us-east-1}
SSO_ACCOUNT_ID=${SSO_ACCOUNT_ID:-857654397073}
SSO_ROLE_NAME=${SSO_ROLE_NAME:-drivers-test-secrets-role}
AWS_REGION=${AWS_REGION:-us-east-1}
export AWS_PROFILE

# Write the profile into a config file owned by this container rather than
# into a mounted ~/.aws, so a run never edits the host's AWS config. Only the
# SSO token cache is shared with the host, and the CLI always reads that from
# $HOME/.aws/sso/cache regardless of AWS_CONFIG_FILE.
AWS_CONFIG_FILE=${AWS_CONFIG_FILE:-/root/aws-config}
export AWS_CONFIG_FILE

SECRETS_DIR=${SECRETS_DIR:-/secrets}
mkdir -p "$SECRETS_DIR"

aws configure set sso_start_url "$SSO_START_URL" --profile "$AWS_PROFILE"
aws configure set sso_region "$SSO_REGION" --profile "$AWS_PROFILE"
aws configure set sso_account_id "$SSO_ACCOUNT_ID" --profile "$AWS_PROFILE"
aws configure set sso_role_name "$SSO_ROLE_NAME" --profile "$AWS_PROFILE"
aws configure set region "$AWS_REGION" --profile "$AWS_PROFILE"
aws configure set output json --profile "$AWS_PROFILE"

# Reuses the cached token if the mounted cache still holds a live session,
# and otherwise prints a verification URL and code.
aws sso login --profile "$AWS_PROFILE" --no-browser

# "--format env" emits "export KEY=VALUE" lines for the access key, secret key
# and session token.
aws configure export-credentials --profile "$AWS_PROFILE" --format env \
    >"$SECRETS_DIR/secrets-export.sh"

echo "wrote credentials to $SECRETS_DIR/secrets-export.sh"

# Verify the exported credentials actually authenticate. This deliberately
# loads them as environment variables instead of passing --profile: resolving
# through the profile would re-read the SSO cache and prove nothing about what
# was written above. Region still comes from AWS_REGION.
#
# Set VERIFY_CREDENTIALS=0 to skip, e.g. when the container has no egress to
# the STS endpoint.
if [ "${VERIFY_CREDENTIALS:-1}" = "1" ]; then
    # shellcheck source=/dev/null
    . "$SECRETS_DIR/secrets-export.sh"

    export AWS_REGION

    echo "verifying exported credentials with sts get-caller-identity..."
    aws sts get-caller-identity
fi

# Optionally fetch AWS Secrets Manager vaults and append them to the same
# file. This is what drivers-evergreen-tools' setup_secrets.py does: each
# vault is a flat JSON object, whose keys are upper-cased and emitted as
# exports. Doing it here means a caller needs neither a Python venv nor a host
# AWS profile — the SSO login above already granted the role that can read
# these vaults.
#
# SECRET_VAULTS is a space-separated list, e.g. "drivers/csfle".
if [ -n "${SECRET_VAULTS:-}" ]; then
    for vault in $SECRET_VAULTS; do
        echo "fetching secrets from vault $vault..."

        aws secretsmanager get-secret-value \
            --secret-id "$vault" \
            --profile "$AWS_PROFILE" \
            --query SecretString \
            --output text |
            jq -r 'to_entries[] | "export \(.key | ascii_upcase)=\"\(.value)\""' \
                >>"$SECRETS_DIR/secrets-export.sh"
    done

    echo "appended vault secrets to $SECRETS_DIR/secrets-export.sh"
fi
