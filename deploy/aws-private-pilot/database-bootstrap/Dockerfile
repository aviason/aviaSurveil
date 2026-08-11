ARG POSTGRES_CLIENT_IMAGE=docker.io/library/postgres@sha256:49b6b7eaf6068472b9cc825f2bd8e0aa03ed34ab55f214a8ecc011f8e624da30

FROM ${POSTGRES_CLIENT_IMAGE} AS database-bootstrap
RUN apk upgrade --no-cache \
    && addgroup -S -g 10001 avia \
    && adduser -S -D -H -u 10001 -G avia avia
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/psql"]
