TARGET=gobot
TS=$(shell date -u +"%F_%T")
TAG=$(shell git tag | sort -V | tail -1)
COMMIT=$(shell git log --oneline | head -1)
VERSION=$(firstword $(COMMIT))
LDFLAGS=-X main.Version=$(TAG) -X main.Revision=git:$(VERSION) -X main.BuildDate=$(TS)
TEST_DB=/tmp/gobot_db_test.sqlite
TEST_CONFIG=/tmp/gobot_config_test.toml
TEST_DB_REPLACED=$(shell echo $(TEST_DB) | sed -e 's/[\/&]/\\&/g')
CONTAINER=docker/container_build.sh
DOCKER_TAG=z0rr0/gobot

# coverage check
# go tool cover -html=coverage.out

all: build

build:
	go build -o $(PWD)/$(TARGET) -ldflags "$(LDFLAGS)"

fmt:
	gofmt -d .

check_fmt:
	@test -z "`gofmt -l .`" || { echo "ERROR: failed gofmt, for more details run - make fmt"; false; }
	@-echo "gofmt successful"

lint: check_fmt
	go vet $(PWD)/...
	golint -set_exit_status $(PWD)/...
	golangci-lint run $(PWD)/...

prepare:
	rm -f $(TEST_DB) $(TEST_CONFIG)
	cat $(PWD)/db.sql | sqlite3 $(TEST_DB)
	cat $(PWD)/config.example.toml | sed -e "s/db.sqlite/$(TEST_DB_REPLACED)/g" > $(TEST_CONFIG)

test: lint prepare
	# go test -v -race -cover -coverprofile=coverage.out -trace trace.out github.com/z0rr0/gobot/serve
	go test -race -cover $(PWD)/...

gh: prepare
	go test -race -cover $(PWD)/...

fuzz:
	go test -fuzz=Fuzz -fuzztime 20s github.com/z0rr0/gobot/cmd

docker: lint clean
	docker build --build-arg LDFLAGS="$(LDFLAGS)" -t $(DOCKER_TAG) .

clean:
	rm -f $(PWD)/$(TARGET)
	find ./ -type f -name "*.out" -delete
