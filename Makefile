.PHONY: test
test:
	go test -v
.PHONY: setproj
setproj:
	export GOPATH=/opt/goproj:/opt/goproj/github.com/tweakmy/fotofona
.PHONY: build
build:
	go build -ldflags "-X main.buildtimestamp=`date -u '+%Y-%m-%d_%I:%M:%S%p'` -X fotofona/main.githash=`git rev-parse HEAD`" -o ./bin/fotofona
.PHONY: etcd
etcd:
	etcd --listen-client-urls=http://localhost:2378 --advertise-client-urls=http://localhost:2378 --listen-peer-urls=http://localhost:2382 --data-dir=test-etcd 
.PHONY: dns
dns:
	coredns -dns.port=8053 -conf=.test-dns/Corefile
