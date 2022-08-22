FROM alpine:3.16
MAINTAINER Alexander Zaytsev "me@axv.email"
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates tzdata sqlite
ADD gobot /bin/gobot
RUN chmod 0755 /bin/gobot
EXPOSE 8082
VOLUME ["/data/gobot/"]
ENTRYPOINT ["gobot"]
CMD ["-config", "/data/gobot/config.toml"]
