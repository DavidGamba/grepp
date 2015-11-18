#!/bin/bash

flags=(-ldflags "-X github.com/davidgamba/grepp/semver.BuildMetadata=`git rev-parse HEAD`")

go install "${flags[@]}" grepp.go
