#!/usr/bin/env bash

CONTAINER="gobotbuilder:latest"
SOURCES="$1"
FLAG="$2"
IDCMD=$(command -v id)
DCMD=$(command -v docker)
PERM="$(${IDCMD} -u ${USER}):$(${IDCMD} -g ${USER})"
TARGET=gobot

if [ -z "$DCMD" ]; then
  echo "docker not found"
  exit 1
fi

pushd "${SOURCES}"/docker || exit 2
docker build -t ${CONTAINER} .
popd || exit 3

cd "${SOURCES}" || exit 4

$DCMD run --rm --user "${PERM}" \
  --volume "${SOURCES}":/usr/app \
  --workdir /usr/app \
  --env GOCACHE=/tmp/.cache \
  ${CONTAINER} go build -o /usr/app/${TARGET} -ldflags "${FLAG}"

if [[ $? -gt 0 ]]; then
  echo "ERROR: build container"
  exit 5
fi
