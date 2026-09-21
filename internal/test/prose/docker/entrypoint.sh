#!/usr/bin/env bash
# Runs an interactive AWS SSO login, then writes the resulting credentials to
# $SECRETS_DIR/secrets-export.sh in the same "export KEY=VALUE" format used by
# drivers-evergreen-tools.
#
# "aws sso login --no-browser" prints a URL and a verification code to stdout;
# the caller is responsible for surfacing those so a human can complete the
# flow. ExportSecrets streams container logs to the test log for that reason.
set -eu

AWS_PROFILE=${AWS_PROFILE:-sso}
export AWS_PROFILE

SECRETS_DIR=${SECRETS_DIR:-/secrets}
mkdir -p "$SECRETS_DIR"

# A profile is configured if it has an SSO start URL. Everything else the
# login needs (region, account, role) is written alongside it by
# "aws configure sso".
if [ -n "$(aws configure get sso_start_url --profile "$AWS_PROFILE" 2>/dev/null || true)" ]; then
    aws sso login --profile "$AWS_PROFILE" --no-browser
else
    # "aws configure sso" prompts for the start URL and then lists the
    # accounts and roles the session grants, so nothing has to be known up
    # front. It needs a TTY, which rules out non-interactive callers.
    if [ "${ALLOW_CONFIGURE_SSO:-1}" != "1" ]; then
        echo "error: profile '$AWS_PROFILE' is not configured for SSO, and this run is" >&2
        echo "non-interactive. Configure it once with:" >&2
        echo "  docker run -it --rm -v \"\$HOME/.aws:/root/.aws\" -v \"\$PWD/out:/secrets\" \\" >&2
        echo "    -e AWS_PROFILE=$AWS_PROFILE aws-sso-login" >&2
        exit 2
    fi

    echo "profile '$AWS_PROFILE' is not configured for SSO; starting aws configure sso"
    aws configure sso --profile "$AWS_PROFILE" --no-browser
fi

# "--format env" emits "export KEY=VALUE" lines for the access key, secret key
# and session token.
aws configure export-credentials --profile "$AWS_PROFILE" --format env \
    >"$SECRETS_DIR/secrets-export.sh"

echo "wrote credentials to $SECRETS_DIR/secrets-export.sh"

# Verify the exported credentials actually authenticate. This deliberately
# loads them as environment variables instead of passing --profile: resolving
# through the profile would re-read the SSO cache and prove nothing about what
# was written above. Region still comes from the profile in the mounted config.
#
# Set VERIFY_CREDENTIALS=0 to skip, e.g. when the container has no egress to
# the STS endpoint.
if [ "${VERIFY_CREDENTIALS:-1}" = "1" ]; then
    # shellcheck source=/dev/null
    . "$SECRETS_DIR/secrets-export.sh"

    echo "verifying exported credentials with sts get-caller-identity..."
    aws sts get-caller-identity
fi
