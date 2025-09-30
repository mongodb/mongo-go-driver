#!/usr/bin/env bash
# Convenience wrapper for profiling and tracing Go tests.
set -eu

# Set the prompt for the select menu.
PS3="Choose a profile type: "

options=("cpu" "mem" "block" "mutex" "trace")

select choice in "${options[@]}"
do
    case $choice in
        "cpu" | "mem" | "block" | "mutex")
            tmpfile=$(mktemp)
            echo "Writing $choice profile to $tmpfile"

            go test -${choice}profile ${tmpfile} "$@"
            go tool pprof -http=: ${tmpfile}
            break
            ;;
        "trace")
            tmpfile=$(mktemp)
            echo "Writing trace to $tmpfile"

            go test -trace ${tmpfile} "$@"
            go tool trace ${tmpfile}
            break
            ;;
        *)
            echo "Invalid option. Please try again."
            ;;
    esac
done
