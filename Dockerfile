ARG GOLANG_VERSION="1.21.4"

FROM golang:${GOLANG_VERSION}-alpine as builder
ARG LDFLAGS
RUN apk --no-cache add ca-certificates tzdata git sqlite gcc libc-dev
WORKDIR /go/src/github.com/z0rr0/gobot
COPY . .
RUN echo "LDFLAGS = $LDFLAGS"
RUN GOOS=linux go build -ldflags "$LDFLAGS" -o ./gobot

FROM alpine:3.18
LABEL org.opencontainers.image.authors="me@axv.email" \
        org.opencontainers.image.url="https://hub.docker.com/r/z0rr0/gobot" \
        org.opencontainers.image.documentation="https://github.com/z0rr0/gobot" \
        org.opencontainers.image.source="https://github.com/z0rr0/gobot" \
        org.opencontainers.image.licenses="GPL-3.0" \
        org.opencontainers.image.title="GoBot" \
        org.opencontainers.image.description="Vk Teams messenger goBot"

COPY --from=builder /go/src/github.com/z0rr0/gobot/gobot /bin/
RUN chmod 0755 /bin/gobot

VOLUME ["/data/gobot/"]
ENTRYPOINT ["/bin/gobot"]
CMD ["-config", "/data/gobot/config.toml"]
