ARG GO_BUILD_IMAGE=golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc
ARG GO_RUNTIME_IMAGE=alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

FROM ${GO_BUILD_IMAGE} AS go-build
ARG SOURCE_DATE_EPOCH=0
ENV CGO_ENABLED=0 \
    SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN mkdir -p /out \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/api ./cmd/api \
    && go build -trimpath -tags canonicaltest -buildvcs=false -ldflags="-s -w -buildid=" -o /out/canonical-test-api ./cmd/api \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/worker ./cmd/worker \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/migrate ./cmd/migrate \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/runtime-database-bootstrap ./cmd/runtime-database-bootstrap \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/curated-foundation-loader ./cmd/curated-foundation-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/prepared-roster-loader ./cmd/prepared-roster-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/approved-aga-catalog-loader ./cmd/approved-aga-catalog-loader

FROM ${GO_RUNTIME_IMAGE} AS runtime-base
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -S -g 10001 avia \
    && adduser -S -D -H -u 10001 -G avia avia \
    && mkdir -p /app \
    && chown 10001:10001 /app
WORKDIR /app
ENV TZ=UTC
USER 10001:10001

FROM runtime-base AS api
COPY --from=go-build --chown=10001:10001 /out/api /app/api
COPY --chown=10001:10001 deploy/local/api/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS canonical-test-api
COPY --from=go-build --chown=10001:10001 /out/canonical-test-api /app/api
COPY --chown=10001:10001 deploy/local/api/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS worker
COPY --from=go-build --chown=10001:10001 /out/worker /app/worker
COPY --chown=10001:10001 deploy/local/worker/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS migration
COPY --from=go-build --chown=10001:10001 /out/migrate /app/migrate
COPY --chown=10001:10001 deploy/local/api/migration-entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS bootstrap
COPY --from=go-build --chown=10001:10001 /out/runtime-database-bootstrap /app/runtime-database-bootstrap
COPY --from=go-build --chown=10001:10001 /out/curated-foundation-loader /app/curated-foundation-loader
COPY --from=go-build --chown=10001:10001 /out/prepared-roster-loader /app/prepared-roster-loader
COPY --from=go-build --chown=10001:10001 /out/approved-aga-catalog-loader /app/approved-aga-catalog-loader
COPY --chown=10001:10001 deliverables/AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip /app/bootstrap/AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip
ENTRYPOINT ["/app/curated-foundation-loader", "load"]
