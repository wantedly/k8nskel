NAME := k8nskel
LDFLAGS := -w -s -extldflags '-static'
SRC := $(shell find . -type f -name '*.go')
DOCKER_IMAGE_NAME := quay.io/wantedly/$(NAME)

.DEFAULT_GOAL := bin/$(NAME)

.PHONY: deps
deps:
	go mod download

.PHONY: bin/$(NAME)
bin/$(NAME): $(SRC)
	CGO_ENABLED=0 go build -tags netgo -installsuffix netgo -ldflags "$(LDFLAGS)" -o bin/$(NAME)

.PHONY: clean
clean:
	rm -fr bin/*

.PHONY: docker-build
docker-build:
	GOOS=linux GOARCH=amd64 $(MAKE) bin/$(NAME)
	docker build -t $(DOCKER_IMAGE_NAME) .

# TODO: Add cross-build target
