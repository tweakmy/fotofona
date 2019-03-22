.PHONY: test
test:
	go test -v
.PHONY: setproj
setproj:
	export GOPATH=/opt/goproj:/opt/goproj/github.com/tweakmy/fotofona
