#!/usr/bin/env bash
# Convenience wrapper for profiling and tracing Go tests.
set -eu

# Create a temporary file to store the profile or trace data. Trap the exit
# signals to clean it up after the script exits (typically via Ctrl-C).
tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT INT TERM

# Set the prompt and options for the select menu.
PS3="Choose a profile type: "
options=("cpu" "mem" "block" "mutex" "trace")

select choice in "${options[@]}"
do
    case $choice in
        "cpu" | "mem" | "block" | "mutex")
            echo "Writing $choice profile to $tmpfile"

            go test -${choice}profile ${tmpfile} "$@"
            go tool pprof -http=: ${tmpfile}
            break
            ;;
        "trace")
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
