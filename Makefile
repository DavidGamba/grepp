BUILD_FLAGS=-ldflags="-X github.com/davidgamba/grepp/semver.BuildMetadata=`git rev-parse HEAD`"

install:
	go install $(BUILD_FLAGS) grepp.go
