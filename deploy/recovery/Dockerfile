ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc
ARG POSTGRES_IMAGE=postgres:17.6-alpine3.22@sha256:ef257d85f76e48da1c64832459b59fcaba1a4dac97bf5d7450c77753542eee94

FROM ${GO_BUILD_IMAGE} AS recovery-build
ARG SOURCE_DATE_EPOCH=0
ENV CGO_ENABLED=0 \
    SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN mkdir -p /out \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/object-backup ./cmd/object-backup \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/recovery-fingerprint ./cmd/recovery-fingerprint

FROM ${POSTGRES_IMAGE} AS postgres-recovery
USER root
RUN apk add --no-cache \
    jq=1.8.1-r0 \
    libcrypto3=3.5.7-r0 \
    libssl3=3.5.7-r0 \
    libxml2=2.13.9-r1 \
    musl=1.2.5-r12 \
    musl-utils=1.2.5-r12 \
    pgbackrest=2.55.1-r0 \
    zlib=1.3.2-r0 \
    && rm -f /usr/local/bin/gosu
COPY --from=recovery-build /out/object-backup /usr/local/bin/object-backup
COPY --from=recovery-build /out/recovery-fingerprint /usr/local/bin/recovery-fingerprint
COPY deploy/recovery/pgbackrest-secret /usr/local/bin/pgbackrest-secret
RUN chmod 0555 \
    /usr/local/bin/object-backup \
    /usr/local/bin/recovery-fingerprint \
    /usr/local/bin/pgbackrest-secret
USER 70:70
