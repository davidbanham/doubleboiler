FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/
COPY ./assets /assets
COPY ./views /views
ADD ./bin/app /app

ENTRYPOINT ["/app"]
