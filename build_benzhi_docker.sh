#!/usr/bin/env sh
set -eu

docker build -f benzhi.Dockerfile -t ygw-go-52-01:local .
