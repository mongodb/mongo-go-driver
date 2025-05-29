#!/usr/bin/env bash
set -eu

VARLIST=(
	PR_TASK
	COMMIT
	DRIVERS_TOOLS
)

# Ensure that all variables required to run the test are set, otherwise throw
# an error.
for VARNAME in "${VARLIST[@]}"; do
	[[ -z "${!VARNAME:-}" ]] && echo "ERROR: $VARNAME not set" && exit 1;
done

case $PR_TASK in
	apply-labels)
		CONFIG=$(pwd)/.github/labeler.yml
		SCRIPT="${DRIVERS_TOOLS}/.evergreen/github_app/apply-labels.sh"
		bash $SCRIPT -l $CONFIG -h $COMMIT -o "mongodb" -n "mongo-go-driver"
		;;
	assign-reviewer)
		CONFIG=$(pwd)/.github/reviewers.txt
		SCRIPT="${DRIVERS_TOOLS}/.evergreen/github_app/assign-reviewer.sh"
		bash $SCRIPT -p $CONFIG -h $COMMIT -o "mongodb" -n "mongo-go-driver"
		;;
	backport-pr)
		bash ${DRIVERS_TOOLS}/.evergreen/github_app/backport-pr.sh mongodb mongo-go-driver $COMMIT
		;;
esac
