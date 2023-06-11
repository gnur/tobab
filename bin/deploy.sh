#!/bin/bash


function log {
  echo "> $(date +%T.%3N) $*"
}
if command -v gdate &> /dev/null; then
  function log {
    echo "> $(gdate +%T.%3N) $*"
  }
fi

dirty=""
if [ -n "$(git status --porcelain)" ]; then
  dirty="-dirty-$(date +%F_%H%M%S)"
fi


version="$(git describe --tags)$dirty"

img="uranus.goat-gecko.ts.net/gnur/tobab:${version}"

log "Building go tobab binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.Version=${version}" -o dist/tobab ./cmd/tobab/

log "Building docker image: $img"
docker build --build-arg VERSION="${version}" --platform=linux/amd64 -t "${img}" .

log "Pushing image"
docker push "${img}"

exit 0

cd /Users/erwindekeijzer/code/src/github.com/gnur/argo/nalo || exit 2
log "Updating gitops dir"
kustomize edit set image "${img}"
git commit -a -m "Update nalo version ${version}"
git push
cd - || exit 2

log "Done!"
