NAME := k8nskel
LDFLAGS := -w -s -extldflags '-static'
SRC := $(shell find . -type f -name '*.go')
DOCKER_IMAGE_NAME := quay.io/wantedly/$(NAME)

.DEFAULT_GOAL := bin/$(NAME)

.PHONY: dep
dep:
ifeq ($(shell command -v dep 2> /dev/null),)
	go get -u github.com/golang/dep/cmd/dep
endif

.PHONY: deps
deps: dep
	dep ensure -v

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
