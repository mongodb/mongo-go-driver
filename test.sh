branch=$(git rev-parse --abbrev-ref HEAD)
lines=$(git --no-pager reflog show --no-abbrev $branch)
sha=$(echo $lines | awk 'END{print}' | awk '{print $1;}')
echo $sha
gorelease -base=$sha