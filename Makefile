VERSION := v0.0.1

BIN_NAME = grpcr
CONTAINER=grpcr
IMPORT_PATH = github.com/vearne/grpcreplay

BUILD_TIME = $(shell date +%Y%m%d%H%M%S)
GITTAG = `git log -1 --pretty=format:"%H"`
LDFLAGS = -ldflags "-s -w -X $(IMPORT_PATH)/consts.GitTag=${GITTAG} -X $(IMPORT_PATH)/consts.BuildTime=${BUILD_TIME} -X $(IMPORT_PATH)/consts.Version=${VERSION}"
SOURCE_PATH = /go/src/github.com/vearne/grpcreplay/

.PHONY: build install release release-linux release-mac docker-img


build:
	go build $(LDFLAGS) -o $(BIN_NAME)