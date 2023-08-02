branch=$(git rev-parse --abbrev-ref HEAD)
lines=$(git --no-pager reflog show --no-abbrev $branch)
sha=$(echo $lines | tail -n 1 | awk '{print $1;}')
gorelease -base=$sha