.PHONY: test
test:
	go test -v
.PHONY: setproj
setproj:
	export GOPATH=/opt/goproj:/opt/goproj/github.com/tweakmy/fotofona
.PHONY: build
build:
	go build -ldflags "-X github.com/tweakmy/fotofona/cmd.buildtimestamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X github.com/tweakmy/fotofona/cmd.githash=`git rev-parse HEAD`" -o ./bin/fotofona
