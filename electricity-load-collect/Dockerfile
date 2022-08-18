FROM alpine:3.16
RUN apk add libc-dev
COPY main /bin/
RUN chmod +x /bin/main
ENTRYPOINT ["/bin/main"]
