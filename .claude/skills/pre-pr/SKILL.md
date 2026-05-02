______________________________________________________________________

## name: pre-pr description: Run pre-PR validation for the mongo-go-driver. Runs the default task target, checks for public API changes, and summarizes what passed or failed. disable-model-invocation: false

Run pre-PR validation for the mongo-go-driver repository. Follow these steps in order:

1. **Run the default task target**: Execute `task` (which runs build, check-license, check-fmt, check-modules, lint, and test-short). Report any failures with the exact error output.

1. **Check for public API changes**: Run `git diff --name-only $(git merge-base HEAD master)..HEAD` and check whether any files under `mongo/`, `bson/`, `event/`, or `tag/` were modified. If yes, remind the user to run `task api-report` and include the output in their PR description.

1. **Check for migration doc updates**: If any user-visible breaking changes were made, remind the user to update `docs/migration-2.0.md`.

1. **Check commit message format**: Verify that commit messages on this branch follow the `GODRIVER-NNNN Short description` format. Flag any that don't.

1. **Summarize**: Report what passed, what failed, and any required follow-up steps (api-report, migration doc, JIRA prefix).

Note: The bundled `mongo-go-driver-pr-review` skill does a deeper review of the code changes themselves — run that after this validation passes.
