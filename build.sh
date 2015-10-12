#!/bin/sh
go build -ldflags "-X github.com/davidgamba/grepp/semver.BuildMetadata=`git rev-parse HEAD`" grepp.go
