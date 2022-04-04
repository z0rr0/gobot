TARGET=gobot
TS=$(shell date -u +"%F_%T")
TAG=$(shell git tag | sort --version-sort | tail -1)
COMMIT=$(shell git log --oneline | head -1)
VERSION=$(firstword $(COMMIT))
FLAG=-X main.Version=$(TAG) -X main.Revision=git:$(VERSION) -X main.BuildDate=$(TS)
TEST_DB=/tmp/gobot_db_test.sqlite

all: build

build:
	go build -o $(PWD)/$(TARGET) -ldflags "$(FLAG)"

fmt:
	gofmt -d .

check_fmt:
	@test -z "`gofmt -l .`" || { echo "ERROR: failed gofmt, for more details run - make fmt"; false; }
	@-echo "gofmt successful"

lint: check_fmt
	go vet $(PWD)/...
	golint -set_exit_status $(PWD)/...
	golangci-lint run $(PWD)/...

test:
	rm -f $(TEST_DB)
	cat $(PWD)/db.sql | sqlite3 $(TEST_DB)
	go test -v -race -cover -coverprofile=coverage.out -trace trace.out github.com/z0rr0/gobot/db
	# go test -race -cover $(PWD)/...

clean:
	rm -f $(PWD)/$(TARGET)
	find ./ -type f -name "*.out" -delete
