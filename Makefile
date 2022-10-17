VERSION := v0.0.1

BIN_NAME = grpcr
CONTAINER = grpcr
IMPORT_PATH = github.com/vearne/grpcreplay

BUILD_TIME = $(shell date +%Y%m%d%H%M%S)
GITTAG = `git log -1 --pretty=format:"%H"`
LDFLAGS = -ldflags "-s -w -X $(IMPORT_PATH)/consts.GitTag=${GITTAG} -X $(IMPORT_PATH)/consts.BuildTime=${BUILD_TIME} -X $(IMPORT_PATH)/consts.Version=${VERSION}"
SOURCE_PATH = /go/src/github.com/vearne/grpcreplay/

.PHONY: build install release release-linux-arm64 release-mac-arm64 docker-img


build:
	go build $(LDFLAGS) -o $(BIN_NAME)

#release: release-linux-amd64 release-mac-arm64

release-linux-amd64:
	docker run -v `pwd`:$(SOURCE_PATH) -it -e GOOS=linux -e GOARCH=amd64 $(CONTAINER) go build $(LDFLAGS) -o $(BIN_NAME)
	tar -zcvf $(BIN_NAME)-$(VERSION)-linux-amd64.tar.gz ./$(BIN_NAME)
	rm $(BIN_NAME)

release-mac-arm64:
	env GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_NAME)
	tar -zcvf $(BIN_NAME)-$(VERSION)-darwin-arm64.tar.gz ./$(BIN_NAME)
	rm $(BIN_NAME)

docker-img:
	docker build --rm -t $(CONTAINER) -f Dockerfile.dev .

clean:
	rm -rf *.pkg
	rm -rf *.zip
	rm -rf *.gz
	rm -rf *.deb
	rm -rf *.rpm