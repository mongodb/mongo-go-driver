#!/usr/bin/env bash
# Stand-in for the AWS CLI, used only by aws-sso-login-fixture.Dockerfile. It
# mimics the two subcommands entrypoint.sh calls and emits obviously fake
# values so a leaked fixture credential is never mistaken for a real one.
set -eu

case "$1 $2" in
"sso login")
    echo "Attempting to automatically open the SSO authorization page..."
    echo "https://example.awsapps.com/start/#/device"
    echo "Then enter the code: FIXTURE-CODE"
    echo "Successfully logged into Start URL: https://example.awsapps.com/start"
    ;;
"configure set")
    # entrypoint.sh writes the hardcoded profile with a series of these. The
    # stub ignores them: nothing downstream of here reads the config.
    ;;
"configure export-credentials")
    echo "export AWS_ACCESS_KEY_ID=fixture-access-key-id"
    echo "export AWS_SECRET_ACCESS_KEY=fixture-secret-access-key"
    echo "export AWS_SESSION_TOKEN=fixture-session-token"
    ;;
"secretsmanager get-secret-value")
    # entrypoint.sh pipes this through jq, so it has to be a flat JSON object.
    # Lower-case keys prove the upper-casing in entrypoint.sh happens.
    echo '{"fle_aws_access_key_id": "fixture-fle-access-key-id", "fle_aws_secret_access_key": "fixture-fle-secret"}'
    ;;
"sts get-caller-identity")
    # Only succeeds if entrypoint.sh actually loaded the exported credentials
    # into the environment, which is what the real check is testing.
    : "${AWS_ACCESS_KEY_ID:?aws-stub: credentials not present in environment}"

    # Lets a test drive the failure path: the secrets file has been written by
    # this point, so only the exit code distinguishes success from failure.
    if [ "${AWS_PROFILE:-}" = "fixture-fail" ]; then
        echo "An error occurred (ExpiredToken) when calling the GetCallerIdentity operation" >&2
        exit 255
    fi

    echo '{"UserId": "FIXTUREUSERID", "Account": "000000000000", "Arn": "arn:aws:sts::000000000000:assumed-role/fixture/fixture"}'
    ;;
*)
    echo "aws-stub: unexpected command: $*" >&2
    exit 1
    ;;
esac
