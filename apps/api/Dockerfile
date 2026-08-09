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
    && go build -trimpath -tags preproddemo -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-api ./cmd/api \
    && go build -trimpath -tags preproddemo -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-lifecycle-probe ./cmd/preprod-aga-demo-lifecycle-probe \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/worker ./cmd/worker \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/data-feed-worker ./cmd/data-feed-worker \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/data-feed-replay ./cmd/data-feed-replay \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/data-feed-backfill ./cmd/data-feed-backfill \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/data-feed-reconcile ./cmd/data-feed-reconcile \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/scheduler ./cmd/reminder-scheduler \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/migrate ./cmd/migrate \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-data-loader ./cmd/preprod-data-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-canonical-aga-loader ./cmd/preprod-canonical-aga-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-canonical-demo-identity-loader ./cmd/preprod-canonical-demo-identity-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-candidate-demo-loader ./cmd/preprod-aga-candidate-demo-loader \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-role-provisioner ./cmd/preprod-aga-demo-role-provisioner \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-normal-runtime-role-provisioner ./cmd/preprod-normal-runtime-role-provisioner
RUN go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-oidc-fixture ./cmd/preprod-aga-demo-oidc-fixture
RUN go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-workspace-role-provisioner ./cmd/preprod-aga-demo-workspace-role-provisioner \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-workspace-fixture-exporter ./cmd/preprod-aga-demo-workspace-fixture-exporter \
    && go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/preprod-aga-demo-workspace-loader ./cmd/preprod-aga-demo-workspace-loader

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

FROM runtime-base AS preprod-aga-demo-api
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-api /app/api
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-lifecycle-probe /app/preprod-aga-demo-lifecycle-probe
COPY --chown=10001:10001 deploy/local/api/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS worker
COPY --from=go-build --chown=10001:10001 /out/worker /app/worker
COPY --chown=10001:10001 deploy/local/worker/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS data-feed-worker
COPY --from=go-build --chown=10001:10001 /out/data-feed-worker /app/data-feed-worker
ENTRYPOINT ["/app/data-feed-worker"]

FROM runtime-base AS data-feed-replay
COPY --from=go-build --chown=10001:10001 /out/data-feed-replay /app/data-feed-replay
ENTRYPOINT ["/app/data-feed-replay"]

FROM runtime-base AS data-feed-backfill
COPY --from=go-build --chown=10001:10001 /out/data-feed-backfill /app/data-feed-backfill
ENTRYPOINT ["/app/data-feed-backfill"]

FROM runtime-base AS data-feed-reconcile
COPY --from=go-build --chown=10001:10001 /out/data-feed-reconcile /app/data-feed-reconcile
ENTRYPOINT ["/app/data-feed-reconcile"]

FROM runtime-base AS scheduler
COPY --from=go-build --chown=10001:10001 /out/scheduler /app/scheduler
COPY --chown=10001:10001 deploy/local/scheduler/entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS migration
COPY --from=go-build --chown=10001:10001 /out/migrate /app/migrate
COPY --chown=10001:10001 deploy/local/api/migration-entrypoint.sh /app/entrypoint.sh
ENTRYPOINT ["/app/entrypoint.sh"]

FROM runtime-base AS preprod-data-loader
COPY --from=go-build --chown=10001:10001 /out/preprod-data-loader /app/preprod-data-loader
COPY --chown=10001:10001 apps/web/src/parity/legacy-screen-source.json /app/catalog/legacy-screen-source.json
COPY --chown=10001:10001 tests/parity/behavior-ledger.json /app/catalog/behavior-ledger.json
ENTRYPOINT ["/app/preprod-data-loader"]

FROM runtime-base AS preprod-canonical-aga-loader
COPY --from=go-build --chown=10001:10001 /out/preprod-canonical-aga-loader /app/preprod-canonical-aga-loader
ENTRYPOINT ["/app/preprod-canonical-aga-loader"]

FROM runtime-base AS preprod-canonical-demo-identity-loader
COPY --from=go-build --chown=10001:10001 /out/preprod-canonical-demo-identity-loader /app/preprod-canonical-demo-identity-loader
ENTRYPOINT ["/app/preprod-canonical-demo-identity-loader"]

FROM runtime-base AS preprod-aga-candidate-demo-loader
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-candidate-demo-loader /app/preprod-aga-candidate-demo-loader
ENTRYPOINT ["/app/preprod-aga-candidate-demo-loader"]

FROM runtime-base AS preprod-aga-demo-role-provisioner
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-role-provisioner /app/preprod-aga-demo-role-provisioner
ENTRYPOINT ["/app/preprod-aga-demo-role-provisioner"]

FROM runtime-base AS preprod-normal-runtime-role-provisioner
COPY --from=go-build --chown=10001:10001 /out/preprod-normal-runtime-role-provisioner /app/preprod-normal-runtime-role-provisioner
ENTRYPOINT ["/app/preprod-normal-runtime-role-provisioner"]

FROM runtime-base AS preprod-aga-demo-oidc-fixture
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-oidc-fixture /app/preprod-aga-demo-oidc-fixture
ENTRYPOINT ["/app/preprod-aga-demo-oidc-fixture"]

FROM runtime-base AS preprod-aga-demo-workspace-role-provisioner
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-workspace-role-provisioner /app/preprod-aga-demo-workspace-role-provisioner
ENTRYPOINT ["/app/preprod-aga-demo-workspace-role-provisioner"]

FROM runtime-base AS preprod-aga-demo-workspace-fixture-exporter
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-workspace-fixture-exporter /app/preprod-aga-demo-workspace-fixture-exporter
ENTRYPOINT ["/app/preprod-aga-demo-workspace-fixture-exporter"]

FROM runtime-base AS preprod-aga-demo-workspace-loader
COPY --from=go-build --chown=10001:10001 /out/preprod-aga-demo-workspace-loader /app/preprod-aga-demo-workspace-loader
ENTRYPOINT ["/app/preprod-aga-demo-workspace-loader"]
