# Image that performs an interactive AWS SSO login and exports the resulting
# credentials to a directory bind-mounted from the host.
#
# Built and run by prose.ExportSecrets; see internal/test/prose/README.md.
FROM public.ecr.aws/aws-cli/aws-cli:latest

# The AWS CLI image sets "aws" as its entrypoint, so override it with a shell
# script that drives the login and the export.
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENV SECRETS_DIR=/secrets

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
