ARG GOLANG_VERSION="1.20.3"

FROM golang:${GOLANG_VERSION}-alpine as builder
ARG LDFLAGS
RUN apk --no-cache add ca-certificates tzdata git sqlite gcc libc-dev
WORKDIR /go/src/github.com/z0rr0/gobot
COPY . .
RUN echo "LDFLAGS = $LDFLAGS"
RUN GOOS=linux go build -ldflags "$LDFLAGS" -o ./gobot

FROM alpine:3.17
MAINTAINER Alexander Zaitsev "me@axv.email"
COPY --from=builder /go/src/github.com/z0rr0/gobot/gobot /bin/
RUN chmod 0755 /bin/gobot

VOLUME ["/data/gobot/"]
ENTRYPOINT ["/bin/gobot"]
CMD ["-config", "/data/gobot/config.toml"]
