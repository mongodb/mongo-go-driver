# Test fixture for aws-sso-login.Dockerfile. It runs the real entrypoint.sh
# against a stub "aws" binary so ExportSecrets can be tested without real AWS
# credentials or an interactive SSO prompt.
#
# Keep the layout (entrypoint path, SECRETS_DIR) identical to
# aws-sso-login.Dockerfile so the two exercise the same code path.
FROM alpine:3.21

RUN apk add --no-cache bash jq

COPY testdata/aws-stub.sh /usr/local/bin/aws
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/aws /usr/local/bin/entrypoint.sh

ENV SECRETS_DIR=/secrets

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
