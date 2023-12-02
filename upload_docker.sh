#!/usr/bin/env bash

TAG=$(git tag | sort -V | tail -1)
VERSION="${TAG:1}"

echo "version: ${VERSION}"

# tag version
docker tag z0rr0/gobot:latest z0rr0/gobot:"${VERSION}"

# push to docker hub
docker push z0rr0/gobot:"${VERSION}"
docker push z0rr0/gobot:latest
