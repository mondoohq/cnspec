FROM alpine:3.16
RUN apk update &&\
    apk add ca-certificates wget tar &&\
    rm -rf /var/cache/apk/*
COPY cnspec /usr/local/bin
ENTRYPOINT ["cnspec"]
CMD ["help"]