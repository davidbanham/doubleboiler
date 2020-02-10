FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/
ADD zoneinfo.tar.gz /
COPY ./assets /assets
COPY ./views /views
ADD ./bin/app /app

ENTRYPOINT ["/app"]
