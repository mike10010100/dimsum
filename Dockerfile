FROM alpine:3.24

VOLUME /usr/src/config
VOLUME /usr/src/passwords

EXPOSE 8080

RUN apk add --no-cache ca-certificates

COPY dimsum /bin/

ENTRYPOINT ["/bin/dimsum"]

CMD ["--config", "/opt/config/dimsum/config.yaml"]