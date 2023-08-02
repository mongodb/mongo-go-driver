branch=$(git rev-parse --abbrev-ref HEAD)
sha=$(git --no-pager reflog show $branch | tail -n 1 | awk '{print $1;}')
gorelease -base=$sha